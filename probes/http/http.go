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
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/probeutils"
	configpb "github/google.com/cloudprober/probes/http/proto"
)

const (
	maxResponseSizeForMetrics = 128
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name   string
	opts   *options.Options
	c      *configpb.ProbeConf
	l      *logger.Logger
	client *http.Client

	// book-keeping params
	targets  []string
	protocol string
	url      string
}

// probeRunResult captures the results of a single probe run. The way we work with
// stats makes sure that probeRunResult and its fields are not accessed concurrently
// (see documentation with statsKeeper below). That's the reason we use metrics.Int
// types instead of metrics.AtomicInt.
// probeRunResult implements the probeutils.ProbeResult interface.
type probeRunResult struct {
	target     string
	total      metrics.Int
	success    metrics.Int
	latency    metrics.Value
	timeouts   metrics.Int
	respCodes  *metrics.Map
	respBodies *metrics.Map
}

func newProbeRunResult(target string, opts *options.Options) probeRunResult {
	prr := probeRunResult{
		target:     target,
		respCodes:  metrics.NewMap("code", &metrics.Int{}),
		respBodies: metrics.NewMap("resp", &metrics.Int{}),
	}
	if opts.LatencyDist != nil {
		prr.latency = opts.LatencyDist.Clone()
	} else {
		prr.latency = metrics.NewFloat(0)
	}
	return prr
}

// Metrics converts probeRunResult into a slice of the metrics that is suitable for
// working with metrics.EventMetrics. This method is part of the probeutils.ProbeResult
// interface.
func (prr probeRunResult) Metrics() *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("total", &prr.total).
		AddMetric("success", &prr.success).
		AddMetric("latency", prr.latency).
		AddMetric("timeouts", &prr.timeouts).
		AddMetric("resp-code", prr.respCodes).
		AddMetric("resp-body", prr.respBodies)
}

// Target returns the p.target. This method is part of the probeutils.ProbeResult
// interface.
func (prr probeRunResult) Target() string {
	return prr.target
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("no http config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c

	p.targets = p.opts.Targets.List()

	switch p.c.GetProtocol() {
	case configpb.ProbeConf_HTTP:
		p.protocol = "http"
	case configpb.ProbeConf_HTTPS:
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
		Timeout:   p.opts.Timeout,
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

func (p *Probe) runProbe(resultsChan chan<- probeutils.ProbeResult) {

	// Refresh the list of targets to probe.
	p.targets = p.opts.Targets.List()

	wg := sync.WaitGroup{}

	for _, target := range p.targets {
		wg.Add(1)

		// Launch a separate goroutine for each target.
		// Write probe results to the "stats" channel.
		go func(target string, resultsChan chan<- probeutils.ProbeResult) {
			defer wg.Done()
			result := newProbeRunResult(target, p.opts)

			// Prepare HTTP.Request for Client.Do
			host := target
			if p.c.GetResolveFirst() {
				ip, err := p.opts.Targets.Resolve(target, 4) // Support IPv4 for now, should be a config option.
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
				result.total.Inc()
				resp, err := p.client.Do(req)
				latency := time.Since(start)

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
					result.success.Inc()
					result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
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
	resultsChan := make(chan probeutils.ProbeResult, len(p.targets))

	// This function is used by StatsKeeper to get the latest list of targets.
	// TODO: Make p.targets mutex protected as it's read and written by concurrent goroutines.
	targetsFunc := func() []string {
		return p.targets
	}
	go probeutils.StatsKeeper(ctx, "http", p.name, time.Duration(p.c.GetStatsExportIntervalMsec())*time.Millisecond, targetsFunc, resultsChan, dataChan, p.l)

	for _ = range time.Tick(p.opts.Interval) {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}
		p.runProbe(resultsChan)
	}
}
