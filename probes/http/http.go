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
	"bytes"
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
	"github.com/google/cloudprober/probes/probeutils"
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
	method   string
	url      string
}

// probeRunResult captures the results of a single probe run. The way we work with
// stats makes sure that probeRunResult and its fields are not accessed concurrently
// (see documentation with statsKeeper below). That's the reason we use metrics.Int
// types instead of metrics.AtomicInt.
// probeRunResult implements the probeutils.ProbeResult interface.
type probeRunResult struct {
	target            string
	total             metrics.Int
	success           metrics.Int
	latency           metrics.Value
	timeouts          metrics.Int
	respCodes         *metrics.Map
	respBodies        *metrics.Map
	validationFailure *metrics.Map
}

func newProbeRunResult(target string, opts *options.Options) probeRunResult {
	prr := probeRunResult{
		target:            target,
		respCodes:         metrics.NewMap("code", &metrics.Int{}),
		respBodies:        metrics.NewMap("resp", &metrics.Int{}),
		validationFailure: metrics.NewMap("validator", &metrics.Int{}),
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
		AddMetric("resp-body", prr.respBodies).
		AddMetric("validation_failure", prr.validationFailure)
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
		return fmt.Errorf("not http config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c

	p.targets = p.opts.Targets.List()
	p.protocol = strings.ToLower(p.c.GetProtocol().String())
	p.method = p.c.GetMethod().String()
	p.url = p.c.GetRelativeUrl()
	if len(p.url) > 0 && p.url[0] != '/' {
		return fmt.Errorf("Invalid Relative URL: %s, must begin with '/'", p.url)
	}

	if p.c.GetIntegrityCheckPattern() != "" {
		p.l.Warningf("integrity_check_pattern field is now deprecated and doesn't do anything.")
	}

	if p.c.GetRequestsPerProbe() != 1 {
		p.l.Warningf("requests_per_probe field is now deprecated and will be removed in future releases.")
	}

	// Create a transport for our use. This is mostly based on
	// http.DefaultTransport with some timeouts changed.
	// TODO(manugarg): Considering cloning DefaultTransport once
	// https://github.com/golang/go/issues/26013 is fixed.
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   p.opts.Timeout,
			KeepAlive: 30 * time.Second, // TCP keep-alive
			DualStack: true,
		}).DialContext,
		MaxIdleConns:        256, // http.DefaultTransport.MaxIdleConns: 100.
		TLSHandshakeTimeout: p.opts.Timeout,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: p.c.GetDisableCertValidation()},
	}

	// If HTTP keep-alives are not enabled (default), disable HTTP keep-alive in
	// transport.
	if !p.c.GetKeepAlive() {
		transport.DisableKeepAlives = true
	} else {
		// If it's been more than 2 probe intervals since connection was used, close it.
		transport.IdleConnTimeout = 2 * p.opts.Interval
	}

	// Extract source IP from config if present and set in transport.
	if p.c.GetSource() != nil {
		source, err := p.getSourceFromConfig()
		if err != nil {
			return err
		}

		if err := p.setSourceInTransport(transport, source); err != nil {
			return err
		}
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

	return nil
}

// setSourceInTransport sets the provided source IP of the probe in the HTTP Transport.
func (p *Probe) setSourceInTransport(transport *http.Transport, source string) error {
	sourceIP := net.ParseIP(source)
	if sourceIP == nil {
		return fmt.Errorf("invalid source IP: %s", source)
	}
	sourceAddr := net.TCPAddr{
		IP: sourceIP,
	}

	dialer := &net.Dialer{
		LocalAddr: &sourceAddr,
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	transport.DialContext = dialer.DialContext

	return nil
}

// getSourceFromConfig returns the source IP from the config either directly
// or by resolving the network interface to an IP, depending on which is provided.
func (p *Probe) getSourceFromConfig() (string, error) {
	switch p.c.Source.(type) {
	case *configpb.ProbeConf_SourceIp:
		return p.c.GetSourceIp(), nil
	case *configpb.ProbeConf_SourceInterface:
		intf := p.c.GetSourceInterface()
		s, err := probeutils.ResolveIntfAddr(intf)
		if err != nil {
			return "", err
		}
		p.l.Infof("Using %v as source address for interface %s.", s, intf)
		return s, nil
	default:
		return "", fmt.Errorf("unknown source type: %v", p.c.GetSource())
	}
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
func (p *Probe) httpRequest(req *http.Request, result *probeRunResult) {
	start := time.Now()
	result.total.Inc()
	resp, err := p.client.Do(req)
	latency := time.Since(start)

	if err != nil {
		if isClientTimeout(err) {
			p.l.Warningf("Target:%s, URL:%s, http.runProbe: timeout error: %v", req.Host, req.URL.String(), err)
			result.timeouts.Inc()
			return
		}
		p.l.Warningf("Target(%s): client.Get: %v", req.Host, err)
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		p.l.Warningf("Target:%s, URL:%s, http.runProbe: error in reading response from target: %v", req.Host, req.URL.String(), err)
		return
	}

	// Calling Body.Close() allows the TCP connection to be reused.
	resp.Body.Close()
	result.respCodes.IncKey(strconv.FormatInt(int64(resp.StatusCode), 10))

	if p.opts.Validators != nil {
		validationFailed := false

		for name, v := range p.opts.Validators {
			success, err := v.Validate(resp, respBody)
			if err != nil {
				p.l.Errorf("Error while running the validator %s: %v", name, err)
				continue
			}
			if !success {
				result.validationFailure.IncKey(name)
				p.l.Debugf("Target:%s, URL:%s, http.runProbe: validation %s failed.", req.Host, req.URL.String(), name)
				validationFailed = true
			}
		}
		// If any validation failed, return now, leaving the success and latency counters unchanged.
		if validationFailed {
			return
		}
	}

	result.success.Inc()
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	if p.c.GetExportResponseAsMetrics() {
		if len(respBody) <= maxResponseSizeForMetrics {
			result.respBodies.IncKey(string(respBody))
		}
	}
}

func (p *Probe) runProbe(ctx context.Context, resultsChan chan<- probeutils.ProbeResult) {
	// Refresh the list of targets to probe.
	p.targets = p.opts.Targets.List()
	reqCtx, cancelReqCtx := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancelReqCtx()

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
			req, err := http.NewRequest(p.method, url, bytes.NewBufferString(p.c.GetBody()))
			if err != nil {
				p.l.Errorf("Target:%s, URL: %s, http.runProbe: error creating HTTP req: %v", target, url, err)
				return
			}
			// Following line is important only for the cases where we resolve the target first.
			req.Host = target

			for _, header := range p.c.GetHeaders() {
				req.Header.Set(header.GetName(), header.GetValue())
			}

			numRequests := int32(0)
			for {
				p.httpRequest(req.WithContext(reqCtx), &result)

				numRequests++
				if numRequests >= p.c.GetRequestsPerProbe() {
					break
				}
				// Sleep for requests_interval_msec before continuing.
				time.Sleep(time.Duration(p.c.GetRequestsIntervalMsec()) * time.Millisecond)
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
	go probeutils.StatsKeeper(ctx, "http", p.name, time.Duration(p.c.GetStatsExportIntervalMsec())*time.Millisecond, targetsFunc, resultsChan, dataChan, p.l)

	for range time.Tick(p.opts.Interval) {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}
		p.runProbe(ctx, resultsChan)
	}
}
