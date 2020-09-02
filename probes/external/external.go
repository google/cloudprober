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
Package external implements an external probe type for cloudprober.

External probe type executes an external process for actual probing. These probes
can have two modes: "once" and "server". In "once" mode, the external process is
started for each probe run cycle, while in "server" mode, external process is
started only if it's not running already and Cloudprober communicates with it
over stdin/stdout for each probe cycle.

TODO(manugarg): Add a way to test this program. Write another program that
implements the probe server protocol and use that for testing.
*/
package external

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/metrics/payload"
	configpb "github.com/google/cloudprober/probes/external/proto"
	serverpb "github.com/google/cloudprober/probes/external/proto"
	"github.com/google/cloudprober/probes/external/serverutils"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets/endpoint"
	"github.com/google/cloudprober/validators"
)

var (
	// TimeBetweenRequests is the time interval between probe requests for
	// multiple targets. In server mode, probe requests for multiple targets are
	// sent to the same external probe process. Sleeping between requests provides
	// some time buffer for the probe process to dequeue the incoming requests and
	// avoids filling up the communication pipe.
	//
	// Note that this value impacts the effective timeout for a target as timeout
	// is applied for all the targets in aggregate. For example, 100th target in
	// the targets list will have the effective timeout of (timeout - 1ms).
	// TODO(manugarg): Make sure that the last target in the list has an impact of
	// less than 1% on its timeout.
	TimeBetweenRequests = 10 * time.Microsecond
	validLabelRe        = regexp.MustCompile(`@(target|address|port|probe|target\.label\.[^@]+)@`)
)

type result struct {
	total, success    int64
	latency           metrics.Value
	validationFailure *metrics.Map
	payloadMetrics    *metrics.EventMetrics
}

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name    string
	mode    string
	cmdName string
	cmdArgs []string
	opts    *options.Options
	c       *configpb.ProbeConf
	l       *logger.Logger

	// book-keeping params
	labelKeys  map[string]bool // Labels for substitution
	requestID  int32
	cmdRunning bool
	cmdStdin   io.Writer
	cmdStdout  io.ReadCloser
	cmdStderr  io.ReadCloser
	replyChan  chan *serverpb.ProbeReply
	targets    []endpoint.Endpoint
	results    map[string]*result // probe results keyed by targets
	dataChan   chan *metrics.EventMetrics

	// default payload metrics that we clone from to build per-target payload
	// metrics.
	payloadParser *payload.Parser
}

func (p *Probe) updateLabelKeys() {
	p.labelKeys = make(map[string]bool)

	updateLabelKeysFn := func(s string) {
		matches := validLabelRe.FindAllStringSubmatch(s, -1)
		for _, m := range matches {
			if len(m) >= 2 {
				// Pick the match within outer parentheses.
				p.labelKeys[m[1]] = true
			}
		}
	}

	for _, opt := range p.c.GetOptions() {
		updateLabelKeysFn(opt.GetValue())
	}
	for _, arg := range p.cmdArgs {
		updateLabelKeysFn(arg)
	}
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not external probe config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c
	p.replyChan = make(chan *serverpb.ProbeReply)

	cmdParts := strings.Split(p.c.GetCommand(), " ")
	p.cmdName = cmdParts[0]
	p.cmdArgs = cmdParts[1:len(cmdParts)]

	// Figure out labels we are interested in
	p.updateLabelKeys()

	switch p.c.GetMode() {
	case configpb.ProbeConf_ONCE:
		p.mode = "once"
	case configpb.ProbeConf_SERVER:
		p.mode = "server"
	default:
		return fmt.Errorf("invalid mode: %s", p.c.GetMode())
	}

	p.results = make(map[string]*result)

	if !p.c.GetOutputAsMetrics() {
		return nil
	}

	defaultKind := metrics.CUMULATIVE
	if p.c.GetMode() == configpb.ProbeConf_ONCE {
		defaultKind = metrics.GAUGE
	}

	var err error
	p.payloadParser, err = payload.NewParser(p.c.GetOutputMetricsOptions(), "external", p.name, metrics.Kind(defaultKind), p.l)
	if err != nil {
		return fmt.Errorf("error initializing payload metrics: %v", err)
	}

	return nil
}

// substituteLabels replaces occurrences of @label@ with the values from
// labels.  It returns the substituted string and a bool indicating if there
// was a @label@ that did not exist in the labels map.
func substituteLabels(in string, labels map[string]string) (string, bool) {
	if len(labels) == 0 {
		return in, !strings.Contains(in, "@")
	}
	delimiter := "@"
	output := ""
	words := strings.Split(in, delimiter)
	count := len(words)
	foundAll := true
	for j, kwd := range words {
		// Even number of words => just copy out.
		if j%2 == 0 {
			output += kwd
			continue
		}
		// Special case: If there are an odd number of '@' (unbalanced), the last
		// odd index doesn't actually have a closing '@', so we just append it as it
		// is.
		if j == count-1 {
			output += delimiter
			output += kwd
			continue
		}

		// Special case: "@@" => "@"
		if kwd == "" {
			output += delimiter
			continue
		}

		// Finally, the labels.
		replace, ok := labels[kwd]
		if ok {
			output += replace
			continue
		}

		// Nothing - put the token back in.
		foundAll = false
		output += delimiter
		output += kwd
		output += delimiter
	}
	return output, foundAll
}

func (p *Probe) startCmdIfNotRunning(startCtx context.Context) error {
	// Start external probe command if it's not running already. Note that here we
	// are trusting the cmdRunning to be set correctly. It can be false for 3 reasons:
	// 1) This is the first call and the process has actually never been started.
	// 2) cmd.Start() started the process but still returned an error.
	// 3) cmd.Wait() returned incorrectly, while the process was still running.
	//
	// 2 or 3 should never happen as per design, but managing processes can be tricky.
	// Documenting here to help with debugging if we run into an issue.
	if p.cmdRunning {
		return nil
	}
	p.l.Infof("Starting external command: %s %s", p.cmdName, strings.Join(p.cmdArgs, " "))
	cmd := exec.CommandContext(startCtx, p.cmdName, p.cmdArgs...)
	var err error
	if p.cmdStdin, err = cmd.StdinPipe(); err != nil {
		return err
	}
	if p.cmdStdout, err = cmd.StdoutPipe(); err != nil {
		return err
	}
	if p.cmdStderr, err = cmd.StderrPipe(); err != nil {
		return err
	}

	go func() {
		scanner := bufio.NewScanner(p.cmdStderr)
		for scanner.Scan() {
			p.l.Warningf("Stderr of %s: %s", cmd.Path, scanner.Text())
		}
	}()

	if err = cmd.Start(); err != nil {
		p.l.Errorf("error while starting the cmd: %s %s. Err: %v", cmd.Path, cmd.Args, err)
		return fmt.Errorf("error while starting the cmd: %s %s. Err: %v", cmd.Path, cmd.Args, err)
	}

	doneChan := make(chan struct{})
	// This goroutine waits for the process to terminate and sets cmdRunning to
	// false when that happens.
	go func() {
		err := cmd.Wait()
		close(doneChan)
		p.cmdRunning = false

		// Spare logging error message if killed explicitly.
		select {
		case <-startCtx.Done():
			return
		}

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				p.l.Errorf("external probe process died with the status: %s. Stderr: %s", exitErr.Error(), string(exitErr.Stderr))
			}
		}
	}()
	go p.readProbeReplies(doneChan)
	p.cmdRunning = true
	return nil
}

func (p *Probe) readProbeReplies(done chan struct{}) error {
	bufReader := bufio.NewReader(p.cmdStdout)
	// Start a background goroutine to read probe replies from the probe server
	// process's stdout and put them on the probe's replyChan. Note that replyChan
	// is a one element channel. Idea is that we won't need buffering other than
	// the one provided by Unix pipes.
	for {
		select {
		case <-done:
			return nil
		default:
		}
		rep, err := serverutils.ReadProbeReply(bufReader)
		if err != nil {
			// Return if external probe process pipe has closed. We get:
			//  io.EOF: when other process has closed the pipe.
			//  os.ErrClosed: when we have closed the pipe (through cmd.Wait()).
			// *os.PathError: deferred close of the pipe.
			_, isPathError := err.(*os.PathError)
			if err == os.ErrClosed || err == io.EOF || isPathError {
				p.l.Errorf("External probe process pipe is closed. Err: %s", err.Error())
				return err
			}
			p.l.Errorf("Error reading probe reply: %s", err.Error())
			continue
		}
		p.replyChan <- rep
	}

}

func (p *Probe) defaultMetrics(target string, result *result) *metrics.EventMetrics {
	em := metrics.NewEventMetrics(time.Now()).
		AddMetric("success", metrics.NewInt(result.success)).
		AddMetric("total", metrics.NewInt(result.total)).
		AddMetric("latency", result.latency).
		AddLabel("ptype", "external").
		AddLabel("probe", p.name).
		AddLabel("dst", target)

	for _, al := range p.opts.AdditionalLabels {
		em.AddLabel(al.KeyValueForTarget(target))
	}

	if p.opts.Validators != nil {
		em.AddMetric("validation_failure", result.validationFailure)
	}

	return em
}

func (p *Probe) labels(ep endpoint.Endpoint) map[string]string {
	labels := make(map[string]string)
	if p.labelKeys["probe"] {
		labels["probe"] = p.name
	}
	if p.labelKeys["target"] {
		labels["target"] = ep.Name
	}
	if p.labelKeys["port"] {
		labels["port"] = strconv.Itoa(ep.Port)
	}
	if p.labelKeys["address"] {
		addr, err := p.opts.Targets.Resolve(ep.Name, p.opts.IPVersion)
		if err != nil {
			p.l.Warningf("Targets.Resolve(%v, %v) failed: %v ", ep.Name, p.opts.IPVersion, err)
		} else if !addr.IsUnspecified() {
			labels["address"] = addr.String()
		}
	}
	for lk, lv := range ep.Labels {
		k := "target.label." + lk
		if p.labelKeys[k] {
			labels[k] = lv
		}
	}
	return labels
}

func (p *Probe) sendRequest(requestID int32, ep endpoint.Endpoint) error {
	req := &serverpb.ProbeRequest{
		RequestId: proto.Int32(requestID),
		TimeLimit: proto.Int32(int32(p.opts.Timeout / time.Millisecond)),
		Options:   []*serverpb.ProbeRequest_Option{},
	}
	for _, opt := range p.c.GetOptions() {
		value, found := substituteLabels(opt.GetValue(), p.labels(ep))
		if !found {
			p.l.Warningf("Missing substitution in option %q", value)
		}
		req.Options = append(req.Options, &serverpb.ProbeRequest_Option{
			Name:  opt.Name,
			Value: proto.String(value),
		})
	}

	p.l.Debugf("Sending a probe request %v to the external probe server for target %v", requestID, ep.Name)
	return serverutils.WriteMessage(req, p.cmdStdin)
}

type requestInfo struct {
	target    string
	timestamp time.Time
}

// probeStatus captures the single probe status. It's only used by runProbe
// functions to pass a probe's status to processProbeResult method.
type probeStatus struct {
	target  string
	success bool
	latency time.Duration
	payload string
}

func (p *Probe) processProbeResult(ps *probeStatus, result *result) {
	if ps.success && p.opts.Validators != nil {
		failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{ResponseBody: []byte(ps.payload)}, result.validationFailure, p.l)

		// If any validation failed, log and set success to false.
		if len(failedValidations) > 0 {
			p.l.Debug("Target:", ps.target, " failed validations: ", strings.Join(failedValidations, ","), ".")
			ps.success = false
		}
	}

	if ps.success {
		result.success++
		result.latency.AddFloat64(ps.latency.Seconds() / p.opts.LatencyUnit.Seconds())
	}

	em := p.defaultMetrics(ps.target, result)
	p.opts.LogMetrics(em)
	p.dataChan <- em

	// If probe is configured to use the external process output (or reply payload
	// in case of server probe) as metrics.
	if p.c.GetOutputAsMetrics() {
		result.payloadMetrics = p.payloadParser.PayloadMetrics(result.payloadMetrics, ps.payload, ps.target)
		p.opts.LogMetrics(result.payloadMetrics)
		p.dataChan <- result.payloadMetrics
	}
}

func (p *Probe) runServerProbe(ctx, startCtx context.Context) {
	requests := make(map[int32]requestInfo)
	var requestsMu sync.RWMutex
	doneChan := make(chan struct{})

	if err := p.startCmdIfNotRunning(startCtx); err != nil {
		p.l.Error(err.Error())
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Read probe replies until we have no outstanding requests or context has
		// run out.
		for {
			_, ok := <-doneChan
			if !ok {
				// It is safe to access requests without lock here as it won't be accessed
				// by the send loop after doneChan is closed.
				p.l.Debugf("Number of outstanding requests: %d", len(requests))
				if len(requests) == 0 {
					return
				}
			}
			select {
			case <-ctx.Done():
				p.l.Error(ctx.Err().Error())
				return
			case rep := <-p.replyChan:
				requestsMu.Lock()
				reqInfo, ok := requests[rep.GetRequestId()]
				if ok {
					delete(requests, rep.GetRequestId())
				}
				requestsMu.Unlock()
				if !ok {
					// Not our reply, could be from the last timed out probe.
					p.l.Warningf("Got a reply that doesn't match any outstading request: Request id from reply: %v. Ignoring.", rep.GetRequestId())
					continue
				}
				success := true
				if rep.GetErrorMessage() != "" {
					p.l.Errorf("Probe for target %v failed with error message: %s", reqInfo.target, rep.GetErrorMessage())
					success = false
				}
				p.processProbeResult(&probeStatus{
					target:  reqInfo.target,
					success: success,
					latency: time.Since(reqInfo.timestamp),
					payload: rep.GetPayload(),
				}, p.results[reqInfo.target])
			}
		}
	}()

	// Send probe requests
	for _, target := range p.targets {
		p.requestID++
		p.results[target.Name].total++
		requestsMu.Lock()
		requests[p.requestID] = requestInfo{
			target:    target.Name,
			timestamp: time.Now(),
		}
		requestsMu.Unlock()
		p.sendRequest(p.requestID, target)
		time.Sleep(TimeBetweenRequests)
	}

	// Send signal to receiver loop that we are done sending request.
	close(doneChan)

	// Wait for receiver goroutine to exit.
	wg.Wait()
}

// runCommand encapsulates command executor in a variable so that we can
// override it for testing.
var runCommand = func(ctx context.Context, cmd string, args []string) ([]byte, error) {
	return exec.CommandContext(ctx, cmd, args...).Output()
}

func (p *Probe) runOnceProbe(ctx context.Context) {
	var wg sync.WaitGroup

	for _, target := range p.targets {
		wg.Add(1)
		go func(target endpoint.Endpoint, result *result) {
			defer wg.Done()
			args := make([]string, len(p.cmdArgs))
			for i, arg := range p.cmdArgs {
				res, found := substituteLabels(arg, p.labels(target))
				if !found {
					p.l.Warningf("Substitution not found in %q", arg)
				}
				args[i] = res
			}

			p.l.Infof("Running external command: %s %s", p.cmdName, strings.Join(args, " "))
			result.total++
			startTime := time.Now()
			b, err := runCommand(ctx, p.cmdName, args)

			success := true
			if err != nil {
				success = false
				if exitErr, ok := err.(*exec.ExitError); ok {
					p.l.Errorf("external probe process died with the status: %s. Stderr: %s", exitErr.Error(), exitErr.Stderr)
				} else {
					p.l.Errorf("Error executing the external program. Err: %v", err)
				}
			}

			p.processProbeResult(&probeStatus{
				target:  target.Name,
				success: success,
				latency: time.Since(startTime),
				payload: string(b),
			}, result)
		}(target, p.results[target.Name])
	}
	wg.Wait()
}

func (p *Probe) updateTargets() {
	p.targets = p.opts.Targets.ListEndpoints()

	for _, target := range p.targets {
		if _, ok := p.results[target.Name]; ok {
			continue
		}

		var latencyValue metrics.Value
		if p.opts.LatencyDist != nil {
			latencyValue = p.opts.LatencyDist.Clone()
		} else {
			latencyValue = metrics.NewFloat(0)
		}

		p.results[target.Name] = &result{
			latency:           latencyValue,
			validationFailure: validators.ValidationFailureMap(p.opts.Validators),
		}

		for _, al := range p.opts.AdditionalLabels {
			al.UpdateForTarget(target.Name, target.Labels)
		}
	}
}

func (p *Probe) runProbe(startCtx context.Context) {
	probeCtx, cancelFunc := context.WithTimeout(startCtx, p.opts.Timeout)
	defer cancelFunc()

	p.updateTargets()

	if p.mode == "server" {
		p.runServerProbe(probeCtx, startCtx)
	} else {
		p.runOnceProbe(probeCtx)
	}
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(startCtx context.Context, dataChan chan *metrics.EventMetrics) {
	p.dataChan = dataChan

	ticker := time.NewTicker(p.opts.Interval)
	defer ticker.Stop()

	for range ticker.C {
		// Don't run another probe if context is canceled already.
		select {
		case <-startCtx.Done():
			return
		default:
		}

		p.runProbe(startCtx)
	}
}
