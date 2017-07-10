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

// Package http implements HTTP probe type.
package http

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/utils"
)

const (
	maxResponseSizeForMetrics = 128
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name     string
	tgts     targets.Targets
	interval time.Duration
	timeout  time.Duration
	c        *ProbeConf
	l        *logger.Logger
	client   *http.Client

	// book-keeping params
	targets  []string
	protocol string
	url      string
}

// probeRunResult captures the results of a single probe run. The way we work with
// stats makes sure that probeRunResult and its fields are not accessed concurrently
// (see documentation with statsKeeper below). That's the reason we use metrics.Int
// types instead of metrics.AtomicInt.
// probeRunResult implements the utils.ProbeResult interface.
type probeRunResult struct {
	target     string
	sent       metrics.Int
	rcvd       metrics.Int
	rtt        metrics.Int // microseconds
	timeouts   metrics.Int
	respCodes  *metrics.Map
	respBodies *metrics.Map
}

func newProbeRunResult(target string) probeRunResult {
	prr := probeRunResult{
		target:     target,
		respCodes:  metrics.NewMap("code", &metrics.Int{}),
		respBodies: metrics.NewMap("resp", &metrics.Int{}),
	}
	// Borgmon and friends expect results to be in milliseconds. We should
	// change expectations at the Borgmon end once the transition to the new
	// metrics model is complete.
	prr.rtt.Str = func(i int64) string {
		return fmt.Sprintf("%.3f", float64(i)/1000)
	}
	return prr
}

// Metrics converts probeRunResult into a slice of the metrics that is suitable for
// working with metrics.EventMetrics. This method is part of the utils.ProbeResult
// interface.
func (prr probeRunResult) Metrics() *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("sent", &prr.sent).
		AddMetric("rcvd", &prr.rcvd).
		AddMetric("rtt", &prr.rtt).
		AddMetric("timeouts", &prr.timeouts).
		AddMetric("resp-code", prr.respCodes).
		AddMetric("resp-body", prr.respBodies)
}

// Target returns the p.target. This method is part of the utils.ProbeResult
// interface.
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
		return fmt.Errorf("no http config")
	}
	p.name = name
	p.tgts = tgts
	p.interval = interval
	p.timeout = timeout
	p.c = c
	p.l = l

	p.targets = p.tgts.List()

	switch p.c.GetProtocol() {
	case ProbeConf_HTTP:
		p.protocol = "http"
	case ProbeConf_HTTPS:
		p.protocol = "https"
	default:
		p.l.Errorf("Invalid Protocol: %s", p.c.GetProtocol())
	}

	p.url = p.c.GetRelativeUrl()
	if len(p.url) > 0 && p.url[0] != '/' {
		p.l.Errorf("Invalid Relative URL: %s, must begin with '/'", p.url)
	}

	// Needs to be non-nil so we can set parameters on it.
	transport := http.DefaultTransport

	// Keep idle connections open until we explicitly close them.
	// This allows us to send multiple requests over the same connection.
	transport.(*http.Transport).MaxIdleConnsPerHost = 1

	// Clients are safe for concurrent use by multiple goroutines.
	p.client = &http.Client{
		Transport: transport,
		Timeout:   p.timeout,
	}

	return nil
}

// Return true if the underlying error indicates a http.Client timeout.
//
// Use for errors returned from http.Client methods (Get, Post).
func isClientTimeout(err error) bool {
	if uerr, ok := err.(*url.Error); ok {
		if nerr, ok := uerr.Err.(net.Error); ok && nerr.Timeout() {
			return true
		}
	}
	return false
}

func (p *Probe) runProbe(resultsChan chan<- utils.ProbeResult) {

	// Refresh the list of targets to probe.
	p.targets = p.tgts.List()

	wg := sync.WaitGroup{}

	for _, target := range p.targets {
		wg.Add(1)

		// Launch a separate goroutine for each target.
		// Write probe results to the "stats" channel.
		go func(target string, resultsChan chan<- utils.ProbeResult) {
			defer wg.Done()
			result := newProbeRunResult(target)

			// Prepare HTTP.Request for Client.Do
			host := target
			if p.c.GetResolveFirst() {
				ip, err := p.tgts.Resolve(target, 4) // Support IPv4 for now, should be a config option.
				if err != nil {
					p.l.Errorf("Target:%s,  http.runProbe: error resolving the target: %v", target, err)
					return
				}
				host = ip.String()
			}
			if p.c.GetPort() != 0 {
				host = fmt.Sprintf("%s:%d", host, p.c.GetPort())
			}
			url := fmt.Sprintf("%s://%s%s", p.protocol, host, p.url)
			req, err := http.NewRequest("GET", url, nil) // nil body
			if err != nil {
				p.l.Errorf("Target:%s, Url: %s, http.runProbe: error creating HTTP req: %v", target, url, err)
				return
			}
			// Following line is important only for the cases where we resolve the target first.
			req.Host = target

			for i := 0; i < int(p.c.GetRequestsPerProbe()); i++ {
				start := time.Now()
				result.sent.Inc()
				resp, err := p.client.Do(req)
				rtt := time.Since(start)

				if err != nil {
					if isClientTimeout(err) {
						p.l.Warningf("Target:%s, Url:%s, http.runProbe: timeout error: %v", target, req.URL.String(), err)
						result.timeouts.Inc()
					} else {
						p.l.Warningf("Target(%s): client.Get: %v", target, err)
					}
				} else {
					respBody, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						p.l.Warningf("Target:%s, Url:%s, http.runProbe: error in reading response from target: %v", target, req.URL.String(), err)
					}
					// Calling Body.Close() allows the TCP connection to be reused.
					resp.Body.Close()
					result.respCodes.IncKey(fmt.Sprintf("%d", resp.StatusCode))
					result.rcvd.Inc()
					result.rtt.IncBy(metrics.NewInt(rtt.Nanoseconds() / 1000))
					if p.c.GetExportResponseAsMetrics() {
						if len(respBody) <= maxResponseSizeForMetrics {
							result.respBodies.IncKey(string(respBody))
						}
					}
				}

				time.Sleep(time.Duration(p.c.GetRequestsIntervalMsec()) * time.Millisecond)
			}
			resultsChan <- result
		}(target, resultsChan)
	}

	// Wait until all probes are done.
	wg.Wait()

	// Don't re-use TCP connections between probe runs.
	if transport, ok := p.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	} else {
		p.l.Warningf("HTTP Client Transport is not http.Transport, should never happen except for testing.")
	}
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	resultsChan := make(chan utils.ProbeResult, len(p.targets))

	// This function is used by StatsKeeper to get the latest list of targets.
	// TODO: Make p.targets mutex protected as it's read and written by concurrent goroutines.
	targetsFunc := func() []string {
		return p.targets
	}
	go utils.StatsKeeper(ctx, "http", p.name, time.Duration(p.c.GetStatsExportIntervalMsec())*time.Millisecond, targetsFunc, resultsChan, dataChan, p.l)

	for _ = range time.Tick(p.interval) {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}
		p.runProbe(resultsChan)
	}
}
