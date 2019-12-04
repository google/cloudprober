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

// Package http implements HTTP probe type.
package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/http/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets/endpoint"
	"github.com/google/cloudprober/validators"
)

const (
	maxResponseSizeForMetrics = 128
	targetsUpdateInterval     = 1 * time.Minute
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name   string
	opts   *options.Options
	c      *configpb.ProbeConf
	l      *logger.Logger
	client *http.Client

	// book-keeping params
	targets      []endpoint.Endpoint
	httpRequests map[string]*http.Request
	results      map[string]*probeResult
	protocol     string
	method       string
	url          string

	// Run counter, used to decide when to update targets or export
	// stats.
	runCnt int64

	// How often to resolve targets (in probe counts), initialized to
	// targetsUpdateInterval / p.opts.Interval. Targets and associated data
	// structures are updated when (runCnt % targetsUpdateFrequency) == 0
	targetsUpdateFrequency int64

	// How often to export metrics (in probe counts), initialized to
	// statsExportInterval / p.opts.Interval. Metrics are exported when
	// (runCnt % statsExportFrequency) == 0
	statsExportFrequency int64
}

type probeResult struct {
	total, success, timeouts int64
	latency                  metrics.Value
	respCodes                *metrics.Map
	respBodies               *metrics.Map
	validationFailure        *metrics.Map
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not http config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c

	p.protocol = strings.ToLower(p.c.GetProtocol().String())
	p.method = p.c.GetMethod().String()

	p.url = p.c.GetRelativeUrl()
	if len(p.url) > 0 && p.url[0] != '/' {
		return fmt.Errorf("Invalid Relative URL: %s, must begin with '/'", p.url)
	}

	// Create a transport for our use. This is mostly based on
	// http.DefaultTransport with some timeouts changed.
	// TODO(manugarg): Considering cloning DefaultTransport once
	// https://github.com/golang/go/issues/26013 is fixed.
	dialer := &net.Dialer{
		Timeout:   p.opts.Timeout,
		KeepAlive: 30 * time.Second, // TCP keep-alive
	}

	if p.opts.SourceIP != nil {
		dialer.LocalAddr = &net.TCPAddr{
			IP: p.opts.SourceIP,
		}
	}

	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         dialer.DialContext,
		MaxIdleConns:        256, // http.DefaultTransport.MaxIdleConns: 100.
		TLSHandshakeTimeout: p.opts.Timeout,
	}

	if p.c.GetDisableCertValidation() {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// If HTTP keep-alives are not enabled (default), disable HTTP keep-alive in
	// transport.
	if !p.c.GetKeepAlive() {
		transport.DisableKeepAlives = true
	} else {
		// If it's been more than 2 probe intervals since connection was used, close it.
		transport.IdleConnTimeout = 2 * p.opts.Interval
	}

	if p.c.GetDisableHttp2() {
		// HTTP/2 is enabled by default if server supports it. Setting TLSNextProto
		// to an empty dict is the only to disable it.
		transport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	}

	// Clients are safe for concurrent use by multiple goroutines.
	p.client = &http.Client{
		Transport: transport,
	}

	p.statsExportFrequency = p.opts.StatsExportInterval.Nanoseconds() / p.opts.Interval.Nanoseconds()
	if p.statsExportFrequency == 0 {
		p.statsExportFrequency = 1
	}

	// Update targets and associated data structures (requests and results) once
	// in Init(). It's also called periodically in Start(), at
	// targetsUpdateInterval.
	p.updateTargets()
	p.targetsUpdateFrequency = targetsUpdateInterval.Nanoseconds() / p.opts.Interval.Nanoseconds()
	if p.targetsUpdateFrequency == 0 {
		p.targetsUpdateFrequency = 1
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

// httpRequest executes an HTTP request and updates the provided result struct.
func (p *Probe) doHTTPRequest(req *http.Request, result *probeResult, resultMu *sync.Mutex) {
	start := time.Now()

	resp, err := p.client.Do(req)
	latency := time.Since(start)

	// Note that we take lock on result object outside of the actual request.
	resultMu.Lock()
	defer resultMu.Unlock()

	result.total++

	if err != nil {
		if isClientTimeout(err) {
			p.l.Warning("Target:", req.Host, ", URL:", req.URL.String(), ", http.doHTTPRequest: timeout error: ", err.Error())
			result.timeouts++
			return
		}
		p.l.Warning("Target:", req.Host, ", URL:", req.URL.String(), ", http.doHTTPRequest: ", err.Error())
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		p.l.Warning("Target:", req.Host, ", URL:", req.URL.String(), ", http.doHTTPRequest: ", err.Error())
		return
	}

	// Calling Body.Close() allows the TCP connection to be reused.
	resp.Body.Close()
	result.respCodes.IncKey(strconv.FormatInt(int64(resp.StatusCode), 10))

	if p.opts.Validators != nil {
		failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{Response: resp, ResponseBody: respBody}, result.validationFailure, p.l)

		// If any validation failed, return now, leaving the success and latency
		// counters unchanged.
		if len(failedValidations) > 0 {
			p.l.Debug("Target:", req.Host, ", URL:", req.URL.String(), ", http.doHTTPRequest: failed validations: ", strings.Join(failedValidations, ","))
			return
		}
	}

	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	if p.c.GetExportResponseAsMetrics() {
		if len(respBody) <= maxResponseSizeForMetrics {
			result.respBodies.IncKey(string(respBody))
		}
	}
}

func (p *Probe) updateTargets() {
	p.targets = p.opts.Targets.ListEndpoints()

	if p.httpRequests == nil {
		p.httpRequests = make(map[string]*http.Request, len(p.targets))
	}

	if p.results == nil {
		p.results = make(map[string]*probeResult, len(p.targets))
	}

	for _, target := range p.targets {
		// Update HTTP request
		req := p.httpRequestForTarget(target)
		if req != nil {
			p.httpRequests[target.Name] = req
		}

		for _, al := range p.opts.AdditionalLabels {
			al.UpdateForTarget(target.Name, target.Labels)
		}

		// Add missing result objects
		if p.results[target.Name] == nil {
			var latencyValue metrics.Value
			if p.opts.LatencyDist != nil {
				latencyValue = p.opts.LatencyDist.Clone()
			} else {
				latencyValue = metrics.NewFloat(0)
			}
			p.results[target.Name] = &probeResult{
				latency:           latencyValue,
				respCodes:         metrics.NewMap("code", metrics.NewInt(0)),
				respBodies:        metrics.NewMap("resp", metrics.NewInt(0)),
				validationFailure: validators.ValidationFailureMap(p.opts.Validators),
			}
		}
	}
}

func (p *Probe) runProbe(ctx context.Context) {
	reqCtx, cancelReqCtx := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancelReqCtx()

	wg := sync.WaitGroup{}
	for _, target := range p.targets {
		req, result := p.httpRequests[target.Name], p.results[target.Name]
		if req == nil {
			continue
		}

		// We launch a separate goroutine for each HTTP request. Since there can be
		// multiple requests per probe per target, we use a mutex to protect access
		// to per-target result object in doHTTPRequest. Note that result object is
		// not accessed concurrently anywhere else -- export of the metrics happens
		// when probe is not running.
		var resultMu sync.Mutex

		for numReq := int32(0); numReq < p.c.GetRequestsPerProbe(); numReq++ {
			wg.Add(1)

			go func(req *http.Request, result *probeResult) {
				defer wg.Done()
				p.doHTTPRequest(req.WithContext(reqCtx), result, &resultMu)
			}(req, result)
		}
	}

	// Wait until all probes are done.
	wg.Wait()
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	ticker := time.NewTicker(p.opts.Interval)
	defer ticker.Stop()

	for ts := range ticker.C {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Update targets if its the turn for that.
		if (p.runCnt % p.targetsUpdateFrequency) == 0 {
			p.updateTargets()
		}
		p.runCnt++

		p.runProbe(ctx)

		if (p.runCnt % p.statsExportFrequency) == 0 {
			for _, target := range p.targets {
				result := p.results[target.Name]
				em := metrics.NewEventMetrics(ts).
					AddMetric("total", metrics.NewInt(result.total)).
					AddMetric("success", metrics.NewInt(result.success)).
					AddMetric("latency", result.latency).
					AddMetric("timeouts", metrics.NewInt(result.timeouts)).
					AddMetric("resp-code", result.respCodes).
					AddMetric("resp-body", result.respBodies).
					AddLabel("ptype", "http").
					AddLabel("probe", p.name).
					AddLabel("dst", target.Name)

				for _, al := range p.opts.AdditionalLabels {
					em.AddLabel(al.KeyValueForTarget(target.Name))
				}

				if p.opts.Validators != nil {
					em.AddMetric("validation_failure", result.validationFailure)
				}

				p.opts.LogMetrics(em)
				dataChan <- em
			}
		}
	}
}
