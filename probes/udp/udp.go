// Copyright 2017 Google Inc.
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/targets"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name     string
	tgts     targets.Targets
	interval time.Duration
	timeout  time.Duration
	c        *ProbeConf
	l        *logger.Logger

	// internal book-keeping
	targets []string
}

// probeRunResult captures the results of a single probe run. The way we work with
// stats makes sure that probeRunResult and its fields are not accessed concurrently
// (see documentation with statsKeeper below). That's the reason we use metrics.Int
// types instead of metrics.AtomicInt.
type probeRunResult struct {
	target   string
	sent     metrics.Int
	rcvd     metrics.Int
	rtt      metrics.Int // microseconds
	timeouts metrics.Int
}

func newProbeRunResult(target string) probeRunResult {
	return probeRunResult{
		target: target,
	}
}

// Target returns the p.target.
func (prr probeRunResult) Target() string {
	return prr.target
}

// Metrics converts probeRunResult into metrics.EventMetrics object
func (prr probeRunResult) Metrics() *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("sent", &prr.sent).
		AddMetric("rcvd", &prr.rcvd).
		AddMetric("rtt", &prr.rtt).
		AddMetric("timeouts", &prr.timeouts)
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, tgts targets.Targets, interval, timeout time.Duration, l *logger.Logger, v interface{}) error {
	if l == nil {
		l = &logger.Logger{}
	}
	c, ok := v.(*ProbeConf)
	if !ok {
		return errors.New("not a UDP config")
	}
	p.name = name
	p.tgts = tgts
	p.interval = interval
	p.timeout = timeout
	p.c = c
	p.l = l
	p.targets = p.tgts.List()

	return nil
}

// Return true if the underlying error indicates a udp.Client timeout.
// In our case, we're using the ReadTimeout- time until response is read.
func isClientTimeout(err error) bool {
	e, ok := err.(*net.OpError)
	return ok && e != nil && e.Timeout()
}

// Ping sends a UDP "ping" to addr.  Probe succeeds if the same data is returned within timeout.
// TODO: Hide this once b/32340835 is fixed, that is, once nobody using it anymore.
func Ping(addr string, timeout time.Duration) (success bool, rtt time.Duration, err error) {
	conn, err := net.DialTimeout("udp", addr, timeout)
	if err != nil {
		return false, rtt, err
	}

	defer conn.Close()

	t := time.Now()
	conn.SetWriteDeadline(t.Add(timeout))
	sendData := []byte(t.Format(time.RFC3339Nano))
	if _, err = conn.Write(sendData); err != nil {
		return false, rtt, err
	}

	conn.SetReadDeadline(time.Now().Add(timeout))

	b := make([]byte, 1024)
	n, err := conn.Read(b)
	rtt = time.Since(t)

	success = bytes.Equal(b[:n], sendData)

	return success, rtt, err
}

// runProbe performs a single probe run. The main thread launches one goroutine
// per target to probe. It manages a sync.WaitGroup and Wait's until all probes
// have finished, then exits the runProbe method.
//
// Each per-target goroutine writes its probe result to a channel connected to
// the stats-keeper goroutine, and then exits. This channel is buffered so that
// the per-target goroutines can exit quickly upon probe completion.
func (p *Probe) runProbe(stats chan<- probeutils.ProbeResult) {
	// Refresh the list of targets to probe.
	p.targets = p.tgts.List()

	wg := sync.WaitGroup{}

	for _, target := range p.targets {
		wg.Add(1)

		// Launch a separate goroutine for each target.
		// Write probe results to the "stats" channel.
		go func(target string, stats chan<- probeutils.ProbeResult) {
			defer wg.Done()

			result := probeRunResult{
				target: target,
			}

			// Verified that each request will use different UDP ports.
			fullTarget := net.JoinHostPort(target, fmt.Sprintf("%d", p.c.GetPort()))
			result.sent.Inc()
			success, rtt, err := Ping(fullTarget, p.timeout)

			if err != nil {
				if isClientTimeout(err) {
					p.l.Warningf("udpPing: Target(%s): Timeout error: %v", fullTarget, err)
					result.timeouts.Inc()
				} else {
					p.l.Warningf("udpPing: Target(%s): %v", fullTarget, err)
				}
			} else {
				if success {
					result.rcvd.Inc()
					result.rtt.IncBy(metrics.NewInt(rtt.Nanoseconds() / 1000))
				} else {
					p.l.Warningf("Target(%s): Response is nil, but error is also nil", fullTarget)
				}
			}

			stats <- result
		}(target, stats)
	}

	// Wait until all probes are done.
	wg.Wait()
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	resultsChan := make(chan probeutils.ProbeResult, len(p.targets))
	targetsFunc := func() []string {
		return p.targets
	}
	go probeutils.StatsKeeper(ctx, "udp", p.name, time.Duration(p.c.GetStatsExportIntervalMsec())*time.Millisecond, targetsFunc, resultsChan, dataChan, p.l)

	for range time.Tick(p.interval) {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}

		p.runProbe(resultsChan)
	}
}
