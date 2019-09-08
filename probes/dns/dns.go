// Copyright 2017-2019 Google Inc.
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
	"strings"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/dns/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/validators"
	"github.com/miekg/dns"
)

// Client provides a DNS client interface for required functionality.
// This makes it possible to mock.
type Client interface {
	Exchange(*dns.Msg, string) (*dns.Msg, time.Duration, error)
	setReadTimeout(time.Duration)
	setSourceIP(net.IP)
}

// ClientImpl is a concrete DNS client that can be instantiated.
type clientImpl struct {
	dns.Client
}

// setReadTimeout allows write-access to the underlying ReadTimeout variable.
func (c *clientImpl) setReadTimeout(d time.Duration) {
	c.ReadTimeout = d
}

// setSourceIP allows write-access to the underlying ReadTimeout variable.
func (c *clientImpl) setSourceIP(ip net.IP) {
	c.Dialer = &net.Dialer{
		LocalAddr: &net.UDPAddr{IP: ip},
	}
}

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
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
	target            string
	total             metrics.Int
	success           metrics.Int
	latency           metrics.Value
	timeouts          metrics.Int
	validationFailure *metrics.Map
}

// Metrics converts probeRunResult into metrics.EventMetrics object
func (prr probeRunResult) Metrics() *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("total", &prr.total).
		AddMetric("success", &prr.success).
		AddMetric("latency", prr.latency).
		AddMetric("timeouts", &prr.timeouts).
		AddMetric("validation_failure", prr.validationFailure)
}

// Target returns the p.target.
func (prr probeRunResult) Target() string {
	return prr.target
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
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
	queryType := p.c.GetQueryType()
	if queryType == configpb.QueryType_NONE || int32(queryType) >= int32(dns.TypeReserved) {
		return fmt.Errorf("dns_probe(%v): invalid query type %v", name, queryType)
	}
	p.msg.SetQuestion(dns.Fqdn(p.c.GetResolvedDomain()), uint16(queryType))

	p.client = new(clientImpl)
	if p.opts.SourceIP != nil {
		p.client.setSourceIP(p.opts.SourceIP)
	}
	// Use ReadTimeout because DialTimeout for UDP is not the RTT.
	p.client.setReadTimeout(p.opts.Timeout)

	return nil
}

// Return true if the underlying error indicates a dns.Client timeout.
// In our case, we're using the ReadTimeout- time until response is read.
func isClientTimeout(err error) bool {
	e, ok := err.(*net.OpError)
	return ok && e != nil && e.Timeout()
}

// validateResponse checks status code and answer section for correctness and
// returns true if the response is valid. In case of validation failures, it
// also updates the result structure.
func (p *Probe) validateResponse(resp *dns.Msg, target string, result *probeRunResult) bool {
	if resp == nil || resp.Rcode != dns.RcodeSuccess {
		p.l.Warningf("Target(%s): error in response %v", target, resp)
		return false
	}

	// Validate number of answers in response.
	// TODO: Move this logic to validators.
	minAnswers := p.c.GetMinAnswers()
	if minAnswers > 0 && uint32(len(resp.Answer)) < minAnswers {
		p.l.Warningf("Target(%s): too few answers - got %d want %d.\n\tAnswerBlock: %v",
			target, len(resp.Answer), minAnswers, resp.Answer)
		return false
	}

	if p.opts.Validators != nil {
		answers := []string{}
		for _, rr := range resp.Answer {
			if rr != nil {
				answers = append(answers, rr.String())
			}
		}
		respBytes := []byte(strings.Join(answers, "\n"))

		failedValidations := validators.RunValidators(p.opts.Validators, nil, respBytes, result.validationFailure, p.l)
		if len(failedValidations) > 0 {
			p.l.Debugf("Target(%s): validators %v failed. Resp: %v", target, failedValidations, answers)
			return false
		}
	}

	return true
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
				target:            target,
				validationFailure: validators.ValidationFailureMap(p.opts.Validators),
			}

			if p.opts.LatencyDist != nil {
				result.latency = p.opts.LatencyDist.Clone()
			} else {
				result.latency = metrics.NewFloat(0)
			}

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
			} else if p.validateResponse(resp, fullTarget, &result) {
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
	// TODO(manugarg): Make p.targets mutex protected as it's read and written by concurrent goroutines.
	targetsFunc := func() []string {
		return p.targets
	}

	go probeutils.StatsKeeper(ctx, "dns", p.name, p.opts.StatsExportInterval, targetsFunc, resultsChan, dataChan, p.opts.LogMetrics, p.l)

	ticker := time.NewTicker(p.opts.Interval)
	defer ticker.Stop()

	for range ticker.C {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}
		p.runProbe(resultsChan)
	}
}
