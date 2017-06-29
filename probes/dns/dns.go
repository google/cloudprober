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
Package dns implements a DNS prober. It sends UDP DNS queries to a list of
targets and reports statistics on queries sent, queries received, and latency
experienced.

This prober uses the DNS library in /third_party/golang/dns/dns to construct,
send, and receive DNS messages. Every message is sent on a different UDP port.
Queries to each target are sent in parallel.
*/
package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/utils"
	"github.com/miekg/dns"
)

// Client provides a DNS client interface for required functionality.
// This makes it possible to mock.
type Client interface {
	Exchange(*dns.Msg, string) (*dns.Msg, time.Duration, error)
	SetReadTimeout(time.Duration)
}

// ClientImpl is a concrete DNS client that can be instantiated.
type ClientImpl struct {
	dns.Client
}

// SetReadTimeout allows write-access to the underlying ReadTimeout variable.
func (c *ClientImpl) SetReadTimeout(d time.Duration) {
	c.ReadTimeout = d
}

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name     string
	tgts     targets.Targets
	interval time.Duration
	timeout  time.Duration
	c        *ProbeConf
	l        *logger.Logger

	// book-keeping params
	targets []string
	msg     *dns.Msg
	client  Client
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
	prr := probeRunResult{
		target: target,
	}
	// Borgmon and friends expect results to be in milliseconds. We should
	// change expectations at the Borgmon end once the transition to the new
	// metrics model is complete.
	prr.rtt.Str = func(i int64) string {
		return fmt.Sprintf("%.3f", float64(i)/1000)
	}
	return prr
}

// Metrics converts probeRunResult into metrics.EventMetrics object
func (prr probeRunResult) Metrics() *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("sent", &prr.sent).
		AddMetric("rcvd", &prr.rcvd).
		AddMetric("rtt", &prr.rtt).
		AddMetric("timeouts", &prr.timeouts)
}

// Target returns the p.target.
func (prr probeRunResult) Target() string {
	return prr.target
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, tgts targets.Targets, interval, timeout time.Duration, l *logger.Logger, v interface{}) error {
	if l == nil {
		l = &logger.Logger{}
	}
	c, ok := v.(*ProbeConf)
	if !ok {
		return errors.New("no dns config")
	}
	p.name = name
	p.tgts = tgts
	p.interval = interval
	p.timeout = timeout
	p.c = c
	p.l = l
	p.targets = p.tgts.List()

	// I believe these objects are safe for concurrent use by multiple goroutines
	// (although the documentation doesn't explicitly say so). It uses locks
	// internally and the underlying net.Conn declares that multiple goroutines
	// may invoke methods on a net.Conn simultaneously.
	p.msg = new(dns.Msg)
	p.msg.SetQuestion(dns.Fqdn(p.c.GetResolvedDomain()), dns.TypeMX)

	p.client = new(ClientImpl)
	// Use ReadTimeout because DialTimeout for UDP is not the RTT.
	p.client.SetReadTimeout(p.timeout)

	return nil
}

// Return true if the underlying error indicates a dns.Client timeout.
// In our case, we're using the ReadTimeout- time until response is read.
func isClientTimeout(err error) bool {
	e, ok := err.(*net.OpError)
	return ok && e != nil && e.Timeout()
}

func (p *Probe) runProbe(resultsChan chan<- utils.ProbeResult) {

	// Refresh the list of targets to probe.
	p.targets = p.tgts.List()

	wg := sync.WaitGroup{}

	for _, target := range p.targets {
		wg.Add(1)

		// Launch a separate goroutine for each target.
		// Write probe results to the "resultsChan" channel.
		go func(target string, resultsChan chan<- utils.ProbeResult) {
			defer wg.Done()

			result := probeRunResult{
				target: target,
			}

			// Verified that each request will use different UDP ports.

			fullTarget := net.JoinHostPort(target, "53")
			result.sent.Inc()
			resp, rtt, err := p.client.Exchange(p.msg, fullTarget)

			if err != nil {
				if isClientTimeout(err) {
					p.l.Warningf("Target(%s): client.Exchange: Timeout error: %v", fullTarget, err)
					result.timeouts.Inc()
				} else {
					p.l.Warningf("Target(%s): client.Exchange: %v", fullTarget, err)
				}
			} else {
				if resp == nil {
					p.l.Warningf("Target(%s): Response is nil, but error is also nil", fullTarget)
				}
				result.rcvd.Inc()
				result.rtt.IncBy(metrics.NewInt(rtt.Nanoseconds() / 1000))
			}

			resultsChan <- result
		}(target, resultsChan)
	}

	// Wait until all probes are done.
	wg.Wait()
}

// Run starts and runs the probe indefinitely.
func (p *Probe) Run(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	resultsChan := make(chan utils.ProbeResult, len(p.targets))

	// This function is used by StatsKeeper to get the latest list of targets.
	// TODO: Make p.targets mutex protected as it's read and written by concurrent goroutines.
	targetsFunc := func() []string {
		return p.targets
	}
	go utils.StatsKeeper(ctx, "dns", p.name, time.Duration(p.c.GetStatsExportIntervalMsec())*time.Millisecond, targetsFunc, resultsChan, dataChan, p.l)

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
