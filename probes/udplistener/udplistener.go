// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package udplistener implements a UDP listener. Given a target list, it listens
for packets from each of the targets and reports number of packets successfully
received in order, lost or delayed. It also uses the probe interval as an
indicator for the number of packets we expect from each target. Use the "udp"
probe as the counterpart with the same targets list and probe interval as the
sender.

Notes:

Each probe has 3 goroutines:
- A recvLoop that keeps handling incoming packets and updates metrics.
- An outputLoop that ticks twice every statsExportInterval and outputs metrics.
- An echoLoop that receives incoming packets from recvLoop over a channel and
  echos back the packets.

- Targets list determines which packet sources are valid sources. It is
  updated in the outputLoop routine.
- We use the probe interval to determine the estimated number of packets that
  should be received. This number is the lower bound of the total number of
	packets "sent" by each source.
*/
package udplistener

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/message"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/probeutils"

	udpsrv "github.com/google/cloudprober/servers/udp"
)

const (
	maxMsgSize           = 65536
	maxTargets           = 1024
	logThrottleThreshold = 10
)

// Probe holds aggregate information about all probe runs.
type Probe struct {
	name     string
	opts     *options.Options
	c        *ProbeConf
	l        *logger.Logger
	conn     *net.UDPConn
	echoMode bool

	// map target name to flow state.
	targets []string
	fsm     *message.FlowStateMap

	// Proccess and output results synchronization.
	mu   sync.Mutex
	errs probeErr
	res  map[string]*probeRunResult
}

// proberErr stores error stats and counters for throttled logging.
type probeErr struct {
	throttleCt     int32
	invalidMsgErrs map[string]string // addr -> error string
	missingTargets map[string]int    // sender -> count
}

// echoMsg is a struct that is passed between rx thread and echo thread.
type echoMsg struct {
	addr   *net.UDPAddr
	bufLen int
	buf    []byte
}

func (p *Probe) logErrs() {
	// atomic inc throttleCt so that we don't grab p.mu.Lock() when not logging.
	newVal := atomic.AddInt32(&p.errs.throttleCt, 1)
	if newVal != int32(logThrottleThreshold) {
		return
	}
	defer atomic.StoreInt32(&p.errs.throttleCt, 0)

	p.mu.Lock()
	defer p.mu.Unlock()

	pe := p.errs
	if len(pe.invalidMsgErrs) > 0 {
		p.l.Warningf("Invalid messages received: %v", pe.invalidMsgErrs)
		pe.invalidMsgErrs = make(map[string]string)
	}
	if len(pe.missingTargets) > 0 {
		p.l.Warningf("Unknown targets sending messages: %v", pe.missingTargets)
		pe.missingTargets = make(map[string]int)
	}
}

// probeRunResult captures the results of a single probe run. The way we work with
// stats makes sure that probeRunResult and its fields are not accessed concurrently
// (see documentation with statsKeeper below). That's the reason we use metrics.Int
// types instead of metrics.AtomicInt.
type probeRunResult struct {
	target  string
	total   metrics.Int
	success metrics.Int
	ipdUS   metrics.Int // inter-packet distance in microseconds
	lost    metrics.Int // lost += (currSeq - prevSeq - 1)
	delayed metrics.Int // delayed += (currSeq < prevSeq)
}

// Target returns the p.target.
func (prr probeRunResult) Target() string {
	return prr.target
}

// Metrics converts probeRunResult into metrics.EventMetrics object
func (prr probeRunResult) Metrics() *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("total", &prr.total).
		AddMetric("success", &prr.success).
		AddMetric("ipd_us", &prr.ipdUS).
		AddMetric("lost", &prr.lost).
		AddMetric("delayed", &prr.delayed)
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*ProbeConf)
	if !ok {
		return fmt.Errorf("not a UDP Listener config: %v", opts.ProbeConf)
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c
	p.echoMode = p.c.GetType() == ProbeConf_ECHO

	if time.Duration(c.GetStatsExportIntervalMsec())*time.Millisecond < p.opts.Interval {
		return fmt.Errorf("statsexportinterval %dms smaller than probe interval %v",
			c.GetStatsExportIntervalMsec(), p.opts.Interval)
	}
	p.fsm = message.NewFlowStateMap()

	conn, err := udpsrv.Listen(int(p.c.GetPort()), p.l)
	if err != nil {
		p.l.Warningf("Opening a listen UDP socket on port %d failed: %v", p.c.GetPort(), err)
		return err
	}
	p.conn = conn

	p.res = make(map[string]*probeRunResult)
	p.errs.invalidMsgErrs = make(map[string]string)
	p.errs.missingTargets = make(map[string]int)
	return nil
}

// cleanup closes the udp socket
func (p *Probe) cleanup() {
	if p.conn != nil {
		p.conn.Close()
	}
}

// initProbeRunResults empties the current probe results objects, updates the
// list of targets and builds a new result object for each target.
func (p *Probe) initProbeRunResults() {
	p.targets = p.opts.Targets.List()
	if p.echoMode && len(p.targets) > maxTargets {
		p.l.Warningf("too many targets (got %d > max %d), responses might be slow.", len(p.targets), maxTargets)
	}

	p.res = make(map[string]*probeRunResult)
	for _, target := range p.targets {
		p.res[target] = &probeRunResult{
			target: target,
		}
	}
}

// processMessage processes an incoming message and updates metrics.
func (p *Probe) processMessage(buf []byte, rxTS time.Time, srcAddr *net.UDPAddr) {
	p.mu.Lock()
	defer p.mu.Unlock()

	msg, err := message.NewMessage(buf)
	if err != nil {
		p.errs.invalidMsgErrs[srcAddr.String()] = err.Error()
		return
	}
	src := msg.Src()
	probeRes, ok := p.res[src]
	if !ok {
		p.errs.missingTargets[src]++
		return
	}

	msgRes := msg.ProcessOneWay(p.fsm, rxTS)
	probeRes.total.Inc()
	if msgRes.Success {
		probeRes.success.Inc()
		probeRes.ipdUS.IncBy(metrics.NewInt(msgRes.InterPktDelay.Nanoseconds() / 1000))
	} else if msgRes.LostCount > 0 {
		probeRes.lost.IncBy(metrics.NewInt(int64(msgRes.LostCount)))
	} else if msgRes.Delayed {
		probeRes.delayed.Inc()
	}
}

// outputResults writes results to the output channel.
func (p *Probe) outputResults(expectedCt int64, stats chan<- probeutils.ProbeResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range p.res {
		delta := expectedCt - r.total.Int64()
		if delta > 0 {
			r.total.AddInt64(delta)
		}
		stats <- *r
	}
	p.initProbeRunResults()
}

func (p *Probe) outputLoop(ctx context.Context, stats chan<- probeutils.ProbeResult) {
	// Use a ticker to control stats output and error logging.
	// ticker should be a multiple of interval between pkts (i.e., p.opts.Interval).
	statsExportInterval := time.Duration(p.c.GetStatsExportIntervalMsec()) * time.Millisecond
	pktsPerExportInterval := int64(statsExportInterval / p.opts.Interval)
	tick := p.opts.Interval
	if pktsPerExportInterval > 1 {
		tick = (statsExportInterval / 2).Round(p.opts.Interval)
	}
	ticker := time.NewTicker(tick)

	// Number of packets in an interval = (timeDelta + interval - 1ns) / interval
	// We add (interval/2 - 1ns) because int64 takes the floor, whereas we want
	// to round the expression.
	lastExport := time.Now()
	roundAdd := p.opts.Interval/2 - time.Nanosecond
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			expectedCt := int64((time.Since(lastExport) + roundAdd) / p.opts.Interval)
			p.outputResults(expectedCt, stats)
			p.logErrs()
			lastExport = time.Now()
		}
	}
}

// echoLoop transmits packets received in the msgChan.
func (p *Probe) echoLoop(ctx context.Context, msgChan chan *echoMsg) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-msgChan:
			n, err := p.conn.WriteToUDP(msg.buf, msg.addr)
			if err == io.EOF { // socket closed. exit the loop.
				return
			}
			if err != nil {
				p.l.Errorf("Error writing echo response to %v: %v", msg.addr, err)
			} else if n < msg.bufLen {
				p.l.Warningf("Reply truncated: sent %d out of %d bytes to %v.", n, msg.bufLen, msg.addr)
			}
		}
	}
}

// recvLoop loops over the listener socket for incoming messages and update stats.
// TODO: Move processMessage to the outputLoop and remove probe mutex.
func (p *Probe) recvLoop(ctx context.Context, echoChan chan<- *echoMsg) {
	conn := p.conn
	// Accommodate the largest UDP message.
	b := make([]byte, maxMsgSize)

	p.initProbeRunResults()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		conn.SetReadDeadline(time.Now().Add(time.Second))
		n, srcAddr, err := conn.ReadFromUDP(b)
		if err != nil {
			p.l.Debugf("Error receiving on UDP socket: %v", err)
			continue
		}
		rxTS := time.Now()
		if p.echoMode {
			e := &echoMsg{
				buf:  make([]byte, n),
				addr: srcAddr,
			}
			copy(e.buf, b[:n])
			echoChan <- e
		}
		p.processMessage(b[:n], rxTS, srcAddr)
	}
}

// probeLoop starts the necessary threads and waits for them to exit.
func (p *Probe) probeLoop(ctx context.Context, resultsChan chan<- probeutils.ProbeResult) {
	var wg sync.WaitGroup

	// Output Loop for metrics
	wg.Add(1)
	go func() {
		p.outputLoop(ctx, resultsChan)
		wg.Done()
	}()

	// Echo loop to respond to incoming messages in echo mode.
	var echoChan chan *echoMsg
	if p.echoMode {
		echoChan = make(chan *echoMsg, maxTargets)
		wg.Add(1)
		go func() {
			p.echoLoop(ctx, echoChan)
			wg.Done()
		}()
	}

	p.recvLoop(ctx, echoChan)
	wg.Wait()
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	resultsChan := make(chan probeutils.ProbeResult, len(p.targets))
	targetsFunc := func() []string {
		return p.targets
	}

	go probeutils.StatsKeeper(ctx, "udp", p.name, time.Duration(p.c.GetStatsExportIntervalMsec())*time.Millisecond, targetsFunc, resultsChan, dataChan, p.l)

	// probeLoop runs forever and returns only when the probe has to exit.
	// So, it is safe to cleanup (in the "Start" function) once probeLoop returns.
	p.probeLoop(ctx, resultsChan)
	p.cleanup()
	return
}
