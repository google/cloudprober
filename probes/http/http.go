// Copyright 2017-2020 The Cloudprober Authors.
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
	"math/rand"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/cloudprober/common/oauth"
	"github.com/google/cloudprober/common/tlsconfig"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/http/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets/endpoint"
	"github.com/google/cloudprober/validators"
	"golang.org/x/oauth2"
)

// DefaultTargetsUpdateInterval defines default frequency for target updates.
// Actual targets update interval is:
// max(DefaultTargetsUpdateInterval, probe_interval)
var DefaultTargetsUpdateInterval = 1 * time.Minute

// maxGapBetweenTargets defines the maximum gap between probe loops for each
// target. Actual gap is either configured or determined by the probe interval
// and number of targets.
const maxGapBetweenTargets = 1 * time.Second

const (
	maxResponseSizeForMetrics = 128
	targetsUpdateInterval     = 1 * time.Minute
	largeBodyThreshold        = bytes.MinRead // 512.
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name   string
	opts   *options.Options
	c      *configpb.ProbeConf
	l      *logger.Logger
	client *http.Client

	// book-keeping params
	targets     []endpoint.Endpoint
	protocol    string
	method      string
	url         string
	oauthTS     oauth2.TokenSource
	bearerToken string

	// Run counter, used to decide when to update targets or export
	// stats.
	runCnt int64

	// How often to resolve targets (in probe counts), it's the minimum of
	targetsUpdateInterval time.Duration

	// How often to export metrics (in probe counts), initialized to
	// statsExportInterval / p.opts.Interval. Metrics are exported when
	// (runCnt % statsExportFrequency) == 0
	statsExportFrequency int64

	// Cancel functions for per-target probe loop
	cancelFuncs map[string]context.CancelFunc
	waitGroup   sync.WaitGroup

	requestBody []byte
}

type probeResult struct {
	total, success, timeouts int64
	connEvent                int64
	latency                  metrics.Value
	respCodes                *metrics.Map
	respBodies               *metrics.Map
	validationFailure        *metrics.Map
}

func (p *Probe) updateOauthToken() {
	if p.oauthTS == nil {
		return
	}

	tok, err := p.oauthTS.Token()
	if err != nil {
		p.l.Error("Error getting OAuth token: ", err.Error(), ". Skipping updating the token.")
	} else {
		if tok.AccessToken != "" {
			p.bearerToken = tok.AccessToken
		} else {
			idToken, ok := tok.Extra("id_token").(string)
			if ok {
				p.bearerToken = idToken
			}
		}
		p.l.Debug("Got OAuth token, len: ", strconv.FormatInt(int64(len(p.bearerToken)), 10), ", expirationTime: ", tok.Expiry.String())
	}
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

	p.requestBody = []byte(p.c.GetBody())

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

	if p.c.GetProxyUrl() != "" {
		url, err := url.Parse(p.c.GetProxyUrl())
		if err != nil {
			return fmt.Errorf("error parsing proxy URL (%s): %v", p.c.GetProxyUrl(), err)
		}
		transport.Proxy = http.ProxyURL(url)
	}

	if p.c.GetDisableCertValidation() || p.c.GetTlsConfig() != nil {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}

		if p.c.GetDisableCertValidation() {
			p.l.Warning("disable_cert_validation is deprecated as of v0.10.6. Instead of this, please use \"tls_config {disable_cert_validation: true}\"")
			transport.TLSClientConfig.InsecureSkipVerify = true
		}

		if p.c.GetTlsConfig() != nil {
			if err := tlsconfig.UpdateTLSConfig(transport.TLSClientConfig, p.c.GetTlsConfig(), false); err != nil {
				return err
			}
		}
	}

	// If HTTP keep-alives are not enabled (default), disable HTTP keep-alive in
	// transport.
	if !p.c.GetKeepAlive() {
		transport.DisableKeepAlives = true
	} else {
		// If it's been more than 2 probe intervals since connection was used, close it.
		transport.IdleConnTimeout = 2 * p.opts.Interval
	}

	if p.c.GetOauthConfig() != nil {
		oauthTS, err := oauth.TokenSourceFromConfig(p.c.GetOauthConfig(), p.l)
		if err != nil {
			return err
		}
		p.oauthTS = oauthTS
		p.updateOauthToken() // This is also called periodically.
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

	p.targets = p.opts.Targets.ListEndpoints()
	p.cancelFuncs = make(map[string]context.CancelFunc, len(p.targets))

	p.targetsUpdateInterval = DefaultTargetsUpdateInterval
	// There is no point refreshing targets before probe interval.
	if p.targetsUpdateInterval < p.opts.Interval {
		p.targetsUpdateInterval = p.opts.Interval
	}
	p.l.Infof("Targets update interval: %v", p.targetsUpdateInterval)

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
func (p *Probe) doHTTPRequest(req *http.Request, targetName string, result *probeResult, resultMu *sync.Mutex) {

	if len(p.requestBody) >= largeBodyThreshold {
		req = req.Clone(req.Context())
		req.Body = ioutil.NopCloser(bytes.NewReader(p.requestBody))
	}

	if p.c.GetKeepAlive() {
		trace := &httptrace.ClientTrace{
			ConnectDone: func(_, addr string, err error) {
				result.connEvent++
				if err != nil {
					p.l.Warning("Error establishing a new connection to: ", addr, ". Err: ", err.Error())
					return
				}
				p.l.Info("Established a new connection to: ", addr)
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	}

	start := time.Now()
	resp, err := p.client.Do(req)
	latency := time.Since(start)

	if resultMu != nil {
		// Note that we take lock on result object outside of the actual request.
		resultMu.Lock()
		defer resultMu.Unlock()
	}

	result.total++

	if err != nil {
		if isClientTimeout(err) {
			p.l.Warning("Target:", targetName, ", URL:", req.URL.String(), ", http.doHTTPRequest: timeout error: ", err.Error())
			result.timeouts++
			return
		}
		p.l.Warning("Target:", targetName, ", URL:", req.URL.String(), ", http.doHTTPRequest: ", err.Error())
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		p.l.Warning("Target:", targetName, ", URL:", req.URL.String(), ", http.doHTTPRequest: ", err.Error())
		return
	}

	p.l.Debug("Target:", targetName, ", URL:", req.URL.String(), ", response: ", string(respBody))

	// Calling Body.Close() allows the TCP connection to be reused.
	resp.Body.Close()
	result.respCodes.IncKey(strconv.FormatInt(int64(resp.StatusCode), 10))

	if p.opts.Validators != nil {
		failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{Response: resp, ResponseBody: respBody}, result.validationFailure, p.l)

		// If any validation failed, return now, leaving the success and latency
		// counters unchanged.
		if len(failedValidations) > 0 {
			p.l.Debug("Target:", targetName, ", URL:", req.URL.String(), ", http.doHTTPRequest: failed validations: ", strings.Join(failedValidations, ","))
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

func (p *Probe) runProbe(ctx context.Context, target endpoint.Endpoint, req *http.Request, result *probeResult) {
	reqCtx, cancelReqCtx := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancelReqCtx()

	if p.c.GetRequestsPerProbe() == 1 {
		p.doHTTPRequest(req.WithContext(reqCtx), target.Name, result, nil)
		return
	}

	// For multiple requests per probe, we launch a separate goroutine for each
	// HTTP request. We use a mutex to protect access to per-target result object
	// in doHTTPRequest. Note that result object is not accessed concurrently
	// anywhere else -- export of metrics happens when probe is not running.
	var resultMu sync.Mutex

	wg := sync.WaitGroup{}
	for numReq := int32(0); numReq < p.c.GetRequestsPerProbe(); numReq++ {
		wg.Add(1)
		go func(req *http.Request, targetName string, result *probeResult) {
			defer wg.Done()
			p.doHTTPRequest(req.WithContext(reqCtx), targetName, result, &resultMu)
		}(req, target.Name, result)
	}
	wg.Wait()
}

func (p *Probe) newResult() *probeResult {
	var latencyValue metrics.Value
	if p.opts.LatencyDist != nil {
		latencyValue = p.opts.LatencyDist.Clone()
	} else {
		latencyValue = metrics.NewFloat(0)
	}
	return &probeResult{
		latency:           latencyValue,
		respCodes:         metrics.NewMap("code", metrics.NewInt(0)),
		respBodies:        metrics.NewMap("resp", metrics.NewInt(0)),
		validationFailure: validators.ValidationFailureMap(p.opts.Validators),
	}
}

func (p *Probe) exportMetrics(ts time.Time, result *probeResult, targetName string, dataChan chan *metrics.EventMetrics) {
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(result.total)).
		AddMetric("success", metrics.NewInt(result.success)).
		AddMetric("latency", result.latency).
		AddMetric("timeouts", metrics.NewInt(result.timeouts)).
		AddMetric("resp-code", result.respCodes).
		AddMetric("resp-body", result.respBodies).
		AddLabel("ptype", "http").
		AddLabel("probe", p.name).
		AddLabel("dst", targetName)

	if p.c.GetKeepAlive() {
		em.AddMetric("connect_event", metrics.NewInt(result.connEvent))
	}

	em.LatencyUnit = p.opts.LatencyUnit

	for _, al := range p.opts.AdditionalLabels {
		em.AddLabel(al.KeyValueForTarget(targetName))
	}

	if p.opts.Validators != nil {
		em.AddMetric("validation_failure", result.validationFailure)
	}

	p.opts.LogMetrics(em)
	dataChan <- em
}

func (p *Probe) startForTarget(ctx context.Context, target endpoint.Endpoint, dataChan chan *metrics.EventMetrics) {
	p.l.Debug("Starting probing for the target ", target.Name)

	// We use this counter to decide when to export stats.
	var runCnt int64

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(target.Name, target.Labels)
	}
	result := p.newResult()
	req := p.httpRequestForTarget(target, nil)

	ticker := time.NewTicker(p.opts.Interval)
	defer ticker.Stop()

	for ts := time.Now(); true; ts = <-ticker.C {
		// Don't run another probe if context is canceled already.
		if ctxDone(ctx) {
			return
		}

		// If request is nil (most likely because target resolving failed or it
		// was an invalid target), skip this probe cycle. Note that request
		// creation gets retried at a regular interval (stats export interval).
		if req != nil {
			p.runProbe(ctx, target, req, result)
		}

		// Export stats if it's the time to do so.
		runCnt++
		if (runCnt % p.statsExportFrequency) == 0 {
			p.exportMetrics(ts, result, target.Name, dataChan)

			// If we are resolving first, this is also a good time to recreate HTTP
			// request in case target's IP has changed.
			if p.c.GetResolveFirst() {
				req = p.httpRequestForTarget(target, nil)
			}
		}
	}
}

func (p *Probe) gapBetweenTargets() time.Duration {
	interTargetGap := time.Duration(p.c.GetIntervalBetweenTargetsMsec()) * time.Millisecond

	// If not configured by user, determine based on probe interval and number of
	// targets.
	if interTargetGap == 0 && len(p.targets) != 0 {
		// Use 1/10th of the probe interval to spread out target groroutines.
		interTargetGap = p.opts.Interval / time.Duration(10*len(p.targets))
	}

	return interTargetGap
}

// updateTargetsAndStartProbes refreshes targets and starts probe loop for
// new targets and cancels probe loops for targets that are no longer active.
// Note that this function is not concurrency safe. It is never called
// concurrently by Start().
func (p *Probe) updateTargetsAndStartProbes(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	p.targets = p.opts.Targets.ListEndpoints()

	p.l.Debugf("Probe(%s) got %d targets", p.name, len(p.targets))

	// updatedTargets is used only for logging.
	updatedTargets := make(map[string]string)
	defer func() {
		if len(updatedTargets) > 0 {
			p.l.Infof("Probe(%s) targets updated: %v", p.name, updatedTargets)
		}
	}()

	activeTargets := make(map[string]endpoint.Endpoint)
	for _, target := range p.targets {
		key := target.Key()
		activeTargets[key] = target
	}

	// Stop probing for deleted targets by invoking cancelFunc.
	for targetKey, cancelF := range p.cancelFuncs {
		if _, ok := activeTargets[targetKey]; ok {
			continue
		}
		cancelF()
		updatedTargets[targetKey] = "DELETE"
		delete(p.cancelFuncs, targetKey)
	}

	gapBetweenTargets := p.gapBetweenTargets()
	var startWaitTime time.Duration

	// Start probe loop for new targets.
	for key, target := range activeTargets {
		// This target is already initialized.
		if _, ok := p.cancelFuncs[key]; ok {
			continue
		}
		updatedTargets[key] = "ADD"

		probeCtx, cancelF := context.WithCancel(ctx)
		p.waitGroup.Add(1)

		go func(target endpoint.Endpoint, waitTime time.Duration) {
			defer p.waitGroup.Done()
			// Wait for wait time + some jitter before starting this probe loop.
			time.Sleep(waitTime + time.Duration(rand.Int63n(gapBetweenTargets.Microseconds()/10))*time.Microsecond)
			p.startForTarget(probeCtx, target, dataChan)
		}(target, startWaitTime)

		startWaitTime += gapBetweenTargets

		p.cancelFuncs[key] = cancelF
	}
}

func ctxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// wait waits for child go-routines (one per target) to clean up.
func (p *Probe) wait() {
	p.waitGroup.Wait()
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	defer p.wait()

	p.updateTargetsAndStartProbes(ctx, dataChan)

	// Do more frequent listing of targets until we get a non-zero list of
	// targets.
	for {
		if ctxDone(ctx) {
			return
		}
		if len(p.targets) != 0 {
			break
		}
		p.updateTargetsAndStartProbes(ctx, dataChan)
		time.Sleep(p.opts.Interval)
	}

	targetsUpdateTicker := time.NewTicker(p.targetsUpdateInterval)
	defer targetsUpdateTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-targetsUpdateTicker.C:
			p.updateOauthToken()
			p.updateTargetsAndStartProbes(ctx, dataChan)
		}
	}
}
