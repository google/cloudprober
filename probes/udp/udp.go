// Copyright 2017-2019 The Cloudprober Authors.
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
Package udp implements a UDP prober. It sends UDP queries to a list of
targets and reports statistics on queries sent, queries received, and latency
experienced.

Queries to each target are sent in parallel.
*/
package udp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/google/cloudprober/common/message"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/probeutils"
	configpb "github.com/google/cloudprober/probes/udp/proto"
	udpsrv "github.com/google/cloudprober/servers/udp"
	"github.com/google/cloudprober/sysvars"
	"github.com/google/cloudprober/targets/endpoint"
)

const (
	maxMsgSize = 65536
	// maxTargets is the maximum number of targets supported by this probe type.
	// If there are more targets, they are pruned from the list to bring targets
	// list under maxTargets.
	// TODO(manugarg): Make it configurable with documentation on its implication
	// on resource consumption.
	maxTargets     = 500
	payloadPattern = "cloudprober"
)

// flow represents a UDP flow.
// Since src address and dst port are constant for a probe, src-port and target
// are sufficient to uniquely identify a flow.
type flow struct {
	srcPort string
	target  string
}

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	src  string
	c    *configpb.ProbeConf
	l    *logger.Logger

	// List of UDP connections to use.
	connList    []*net.UDPConn
	srcPortList []string
	numConn     int32
	runID       uint64
	ipVer       int

	targets []endpoint.Endpoint   // List of targets for a probe iteration.
	res     map[flow]*probeResult // Results by flow.
	fsm     *message.FlowStateMap // Map flow parameters to flow state.
	payload []byte

	// Intermediate buffers of sent and received packets
	sentPackets, rcvdPackets chan packetID
	sPackets, rPackets       []packetID
	highestSeq               map[flow]uint64
	flushIntv                time.Duration
}

// probeResult stores the probe results for a target. The way we work with
// stats makes sure that probeResult and its fields are not accessed concurrently
// That's the reason we use metrics.Int types instead of metrics.AtomicInt.
type probeResult struct {
	total, success, delayed int64
	latency                 metrics.Value
}

// Metrics converts probeResult into metrics.EventMetrics object
func (prr probeResult) eventMetrics(probeName string, opts *options.Options, f flow, c *configpb.ProbeConf) *metrics.EventMetrics {
	var suffix string
	if c.GetExportMetricsByPort() {
		suffix = "-per-port"
	}
	m := metrics.NewEventMetrics(time.Now()).
		AddMetric("total"+suffix, metrics.NewInt(prr.total)).
		AddMetric("success"+suffix, metrics.NewInt(prr.success)).
		AddMetric(opts.LatencyMetricName+suffix, prr.latency.Clone()).
		AddMetric("delayed"+suffix, metrics.NewInt(prr.delayed)).
		AddLabel("ptype", "udp").
		AddLabel("probe", probeName).
		AddLabel("dst", f.target)

	for _, al := range opts.AdditionalLabels {
		m.AddLabel(al.KeyValueForTarget(f.target))
	}

	if c.GetExportMetricsByPort() {
		m.AddLabel("src_port", f.srcPort).
			AddLabel("dst_port", fmt.Sprintf("%d", c.GetPort()))
	}

	return m
}

func (p *Probe) newProbeResult() *probeResult {
	var latVal metrics.Value
	if p.opts.LatencyDist != nil {
		latVal = p.opts.LatencyDist.Clone()
	} else {
		latVal = metrics.NewFloat(0)
	}
	return &probeResult{
		latency: latVal,
	}
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return errors.New("not a UDP config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.src = sysvars.Vars()["hostname"]
	p.c = c
	p.fsm = message.NewFlowStateMap()
	p.res = make(map[flow]*probeResult)

	if p.c.GetPayloadSize() != 0 {
		p.payload = make([]byte, p.c.GetPayloadSize())
		probeutils.PatternPayload(p.payload, []byte(payloadPattern))
	}

	// Initialize intermediate buffers of sent and received packets
	p.flushIntv = 2 * p.opts.Interval
	if p.opts.Timeout > p.opts.Interval {
		p.flushIntv = 2 * p.opts.Timeout
	}

	if p.opts.StatsExportInterval < p.flushIntv {
		return fmt.Errorf("UDP probe: stats_export_interval_msec (%s) is too low. It should be at least twice of the interval (%s) and timeout (%s), whichever is bigger", p.opts.StatsExportInterval, p.opts.Interval, p.opts.Timeout)
	}

	// #send/recv-channel-buffer = #targets * #sources * #probing-intervals-between-flushes
	minChanLen := maxTargets * int(p.c.GetNumTxPorts()) * int(math.Ceil(float64(p.flushIntv/p.opts.Interval)))
	p.l.Infof("Creating sent, rcvd channels of length: %d", 2*minChanLen)
	p.sentPackets = make(chan packetID, 2*minChanLen)
	p.rcvdPackets = make(chan packetID, 2*minChanLen)
	p.highestSeq = make(map[flow]uint64)

	// For one-way connections, we use a pool of sockets.
	wantConn := p.c.GetNumTxPorts()
	triesRemaining := wantConn * 2
	p.numConn = 0
	p.connList = make([]*net.UDPConn, wantConn)
	p.srcPortList = make([]string, wantConn)

	udpAddr := &net.UDPAddr{Port: 0}
	if p.opts.SourceIP != nil {
		udpAddr.IP = p.opts.SourceIP
	}
	p.ipVer = p.opts.IPVersion

	for p.numConn < wantConn && triesRemaining > 0 {
		triesRemaining--
		udpConn, err := udpsrv.Listen(udpAddr, p.l)
		if err != nil {
			p.l.Warningf("Opening UDP socket failed: %v", err)
			continue
		}
		p.l.Infof("UDP socket id %d, addr %v", p.numConn, udpConn.LocalAddr())
		p.connList[p.numConn] = udpConn
		_, p.srcPortList[p.numConn], err = net.SplitHostPort(udpConn.LocalAddr().String())
		if err != nil {
			return err
		}
		p.numConn++
	}
	if p.numConn < wantConn {
		for _, c := range p.connList {
			c.Close()
		}
		return fmt.Errorf("UDP socket creation failed: got %d connections, want %d", p.numConn, wantConn)
	}
	return nil
}

// initProbeRunResults initializes missing probe results objects.
func (p *Probe) initProbeRunResults() error {
	for _, target := range p.targets {
		if !p.c.GetExportMetricsByPort() {
			f := flow{"", target.Name}
			if p.res[f] == nil {
				p.res[f] = p.newProbeResult()
			}
			continue
		}

		for _, srcPort := range p.srcPortList {
			f := flow{srcPort, target.Name}
			if p.res[f] == nil {
				p.res[f] = p.newProbeResult()
			}
		}
	}
	return nil
}

// packetID records attributes of the packets sent and received, by runProbe
// and recvLoop respectively. These packetIDs are communicated over channels
// and are eventually processed by the processPackets() loop (below).
type packetID struct {
	f    flow
	seq  uint64
	txTS time.Time
	rxTS time.Time
}

func (p *Probe) resultsKey(f flow) flow {
	if p.c.GetExportMetricsByPort() {
		return f
	}
	return flow{"", f.target}
}

func (p *Probe) processRcvdPacket(rpkt packetID) {
	p.l.Debugf("rpkt seq: %d, target: %s", rpkt.seq, rpkt.f)
	res, ok := p.res[p.resultsKey(rpkt.f)]
	if !ok {
		return
	}
	latency := rpkt.rxTS.Sub(rpkt.txTS)
	if latency < 0 {
		p.l.Errorf("Got negative time delta %v for flow %v seq %d", latency, rpkt.f, rpkt.seq)
		return
	}
	if latency > p.opts.Timeout {
		p.l.Debugf("Packet delayed. Seq: %d, flow: %v, delay: %v", rpkt.seq, rpkt.f, latency)
		res.delayed++
		return
	}
	res.success++
	res.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
}

func (p *Probe) processSentPacket(spkt packetID) {
	p.l.Debugf("spkt seq: %d, flow: %v", spkt.seq, spkt.f)
	res, ok := p.res[p.resultsKey(spkt.f)]
	if !ok {
		return
	}
	res.total++
}

// processPackets processes packets on the sentPackets and rcvdPackets
// channels. Packets are inserted into a lookup map as soon as they are
// received. At every "statsExportInterval" interval, we go through the maps
// and update the probe results.
func (p *Probe) processPackets() {
	// Process packets that we queued earlier (mostly from the last timeout
	// interval)
	for _, rpkt := range p.rPackets {
		p.processRcvdPacket(rpkt)
	}
	for _, spkt := range p.sPackets {
		p.processSentPacket(spkt)
	}
	p.rPackets = p.rPackets[0:0]
	p.sPackets = p.sPackets[0:0]

	lenRcvdPackets := len(p.rcvdPackets)
	p.l.Debugf("rcvd queue length: %d", lenRcvdPackets)
	lenSentPackets := len(p.sentPackets)
	p.l.Debugf("sent queue length: %d", lenSentPackets)

	now := time.Now()
	for i := 0; i < lenSentPackets; i++ {
		pkt := <-p.sentPackets
		if now.Sub(pkt.txTS) < p.opts.Timeout {
			p.l.Debugf("Inserting spacket (seq %d) for late processing", pkt.seq)
			p.sPackets = append(p.sPackets, pkt)
			continue
		}
		p.processSentPacket(pkt)
		if pkt.seq > p.highestSeq[pkt.f] {
			p.highestSeq[pkt.f] = pkt.seq
		}
	}

	for i := 0; i < lenRcvdPackets; i++ {
		pkt := <-p.rcvdPackets
		if now.Sub(pkt.txTS) < p.opts.Timeout {
			p.l.Debugf("Inserting rpacket (seq %d) for late processing", pkt.seq)
			p.rPackets = append(p.rPackets, pkt)
			continue
		}
		if pkt.seq > p.highestSeq[pkt.f] {
			p.l.Debugf("Inserting rpacket for late processing as seq (%d) > highestSeq (%d)", pkt.seq, p.highestSeq[pkt.f])
			p.rPackets = append(p.rPackets, pkt)
			continue
		}
		p.processRcvdPacket(pkt)
	}
}

// Return true if the underlying error indicates a udp.Client timeout.
// In our case, we're using the ReadTimeout- time until response is read.
func isClientTimeout(err error) bool {
	e, ok := err.(*net.OpError)
	return ok && e != nil && e.Timeout()
}

// recvLoop receives all packets over a UDP socket and updates
// flowStates accordingly.
func (p *Probe) recvLoop(ctx context.Context, conn *net.UDPConn) {
	b := make([]byte, maxMsgSize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		conn.SetReadDeadline(time.Now().Add(p.opts.Timeout))
		msgLen, raddr, err := conn.ReadFromUDP(b)
		if err != nil {
			if !isClientTimeout(err) {
				p.l.Errorf("Receive error on %s (from %v): %v", conn.LocalAddr(), raddr, err)
			}
			continue
		}

		rxTS := time.Now()
		msg, err := message.NewMessage(b[:msgLen])
		if err != nil {
			p.l.Errorf("Incoming message error from %s: %v", raddr, err)
			continue
		}
		select {
		case p.rcvdPackets <- packetID{flow{msg.SrcPort(), msg.Dst()}, msg.Seq(), msg.SrcTS(), rxTS}:
		default:
			p.l.Errorf("rcvdPackets channel full")
		}
	}
}

func (p *Probe) runSingleProbe(f flow, conn *net.UDPConn, maxLen, dstPort int) error {
	ip, err := p.opts.Targets.Resolve(f.target, p.ipVer)
	if err != nil {
		return fmt.Errorf("unable to resolve %s: %v", f.target, err)
	}
	raddr := &net.UDPAddr{
		IP:   ip,
		Port: dstPort,
	}

	flowState := p.fsm.FlowState(p.src, f.srcPort, f.target)
	now := time.Now()
	msg, seq, err := flowState.CreateMessage(now, p.payload, maxLen)
	if err != nil {
		return fmt.Errorf("error creating new message to probe target(%s): %v", f.target, err)
	}

	if _, err := conn.WriteToUDP(msg, raddr); err != nil {
		flowState.WithdrawMessage(seq)
		return fmt.Errorf("unable to send to %s(%v): %v", f.target, raddr, err)
	}
	// Send packet over sentPackets channel
	// May need to make a longer buffer for the channel.
	select {
	case p.sentPackets <- packetID{f, seq, now, time.Time{}}:
		return nil
	default:
		return fmt.Errorf("sentPackets channel full")
	}
}

// runProbe performs a single probe run. The main thread launches one goroutine
// per target to probe. It manages a sync.WaitGroup and Wait's until all probes
// have finished, then exits the runProbe method.
//
// Each per-target goroutine sends a UDP message and on success waits for
// "timeout" duration before exiting. "recvLoop" function is expected to
// capture the responses before "timeout" and the main loop will flush the
// results.
func (p *Probe) runProbe() {
	if len(p.targets) == 0 {
		return
	}
	maxLen := int(p.c.GetMaxLength())
	dstPort := int(p.c.GetPort())

	var packetsPerTarget, initialConn int
	if p.c.GetUseAllTxPortsPerProbe() {
		packetsPerTarget = len(p.connList)
		initialConn = 0
	} else {
		packetsPerTarget = 1
		initialConn = int(p.runID % uint64(len(p.connList)))
	}

	var wg sync.WaitGroup
	wg.Add(len(p.targets) * packetsPerTarget)

	for _, conn := range p.connList {
		conn.SetWriteDeadline(time.Now().Add(p.opts.Interval / 2))
	}
	for _, target := range p.targets {
		for i := 0; i < packetsPerTarget; i++ {
			connID := (initialConn + i) % len(p.connList)
			conn := p.connList[connID]
			go func(conn *net.UDPConn, f flow) {
				defer wg.Done()
				if err := p.runSingleProbe(f, conn, maxLen, dstPort); err != nil {
					p.l.Errorf("Probing %+v failed: %v", f, err)
				}
			}(conn, flow{p.srcPortList[connID], target.Name})
		}
	}
	wg.Wait()
	p.runID++
}

func (p *Probe) updateTargets() {
	p.targets = p.opts.Targets.ListEndpoints()
	if len(p.targets) > maxTargets {
		p.l.Warningf("Number of targets (%d) > maxTargets (%d). Truncating the targets list.", len(p.targets), maxTargets)
		p.targets = p.targets[:maxTargets]
	}
	for _, target := range p.targets {
		for _, al := range p.opts.AdditionalLabels {
			al.UpdateForTarget(target)
		}
	}
	p.initProbeRunResults()
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	p.updateTargets()

	for _, conn := range p.connList {
		go p.recvLoop(ctx, conn)
	}

	probeTicker := time.NewTicker(p.opts.Interval)
	statsExportTicker := time.NewTicker(p.opts.StatsExportInterval)
	flushTicker := time.NewTicker(p.flushIntv)

	for {
		select {
		case <-ctx.Done():
			flushTicker.Stop()
			probeTicker.Stop()
			statsExportTicker.Stop()
			return
		case <-probeTicker.C:
			p.runProbe()
		case <-flushTicker.C:
			p.processPackets()
		case <-statsExportTicker.C:
			for f, result := range p.res {
				em := result.eventMetrics(p.name, p.opts, f, p.c)
				em.LatencyUnit = p.opts.LatencyUnit
				p.opts.LogMetrics(em)
				dataChan <- em
			}
			// Use this opportunity to refresh targets as well.
			p.updateTargets()
		}
	}
}
