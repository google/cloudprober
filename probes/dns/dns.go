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
	"net"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/probeutils"
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
	name string
	opts *options.Options
	c    *ProbeConf
	l    *logger.Logger

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
	total    metrics.Int
	success  metrics.Int
	latency  metrics.Float
	timeouts metrics.Int
}

func newProbeRunResult(target string) probeRunResult {
	return probeRunResult{
		target: target,
	}
}

// Metrics converts probeRunResult into metrics.EventMetrics object
func (prr probeRunResult) Metrics() *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("total", &prr.total).
		AddMetric("success", &prr.success).
		AddMetric("latency", &prr.latency).
		AddMetric("timeouts", &prr.timeouts)
}

// Target returns the p.target.
func (prr probeRunResult) Target() string {
	return prr.target
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*ProbeConf)
	if !ok {
		return errors.New("no dns config")
	}
	p.c = c
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.targets = p.opts.Targets.List()

	// I believe these objects are safe for concurrent use by multiple goroutines
	// (although the documentation doesn't explicitly say so). It uses locks
	// internally and the underlying net.Conn declares that multiple goroutines
	// may invoke methods on a net.Conn simultaneously.
	p.msg = new(dns.Msg)
	p.msg.SetQuestion(dns.Fqdn(p.c.GetResolvedDomain()), dns.TypeMX)

	p.client = new(ClientImpl)
	// Use ReadTimeout because DialTimeout for UDP is not the RTT.
	p.client.SetReadTimeout(p.opts.Timeout)

	return nil
}

// Return true if the underlying error indicates a dns.Client timeout.
// In our case, we're using the ReadTimeout- time until response is read.
func isClientTimeout(err error) bool {
	e, ok := err.(*net.OpError)
	return ok && e != nil && e.Timeout()
}

func (p *Probe) runProbe(resultsChan chan<- probeutils.ProbeResult) {

	// Refresh the list of targets to probe.
	p.targets = p.opts.Targets.List()

	wg := sync.WaitGroup{}

	for _, target := range p.targets {
		wg.Add(1)

		// Launch a separate goroutine for each target.
		// Write probe results to the "resultsChan" channel.
		go func(target string, resultsChan chan<- probeutils.ProbeResult) {
			defer wg.Done()

			result := probeRunResult{
				target: target,
			}

			// Verified that each request will use different UDP ports.

			fullTarget := net.JoinHostPort(target, "53")
			result.total.Inc()
			resp, latency, err := p.client.Exchange(p.msg, fullTarget)

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
				result.success.Inc()
				result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
			}

			resultsChan <- result
		}(target, resultsChan)
	}

	// Wait until all probes are done.
	wg.Wait()
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	resultsChan := make(chan probeutils.ProbeResult, len(p.targets))

	// This function is used by StatsKeeper to get the latest list of targets.
	// TODO: Make p.targets mutex protected as it's read and written by concurrent goroutines.
	targetsFunc := func() []string {
		return p.targets
	}
	go probeutils.StatsKeeper(ctx, "dns", p.name, time.Duration(p.c.GetStatsExportIntervalMsec())*time.Millisecond, targetsFunc, resultsChan, dataChan, p.l)

	for range time.Tick(p.opts.Interval) {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}
		p.runProbe(resultsChan)
	}
}
