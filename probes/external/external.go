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
Package external implements an external probe type for cloudprober.

External probe type executes an external process for actual probing. These probes
can have two modes: "once" and "server". In "once" mode, the external process is
started for each probe run cycle, while in "server" mode, external process is
started only if it's not running already and Cloudprober communicates with it
over stdin/stdout for each probe cycle.

TODO: Add a way to test this program. Write another program that
implements the probe server protocol and use that for testing.
*/
package external

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/external/serverutils"
	"github.com/google/cloudprober/probes/options"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name    string
	mode    string
	cmdName string
	cmdArgs []string
	opts    *options.Options
	c       *ProbeConf
	l       *logger.Logger
	ipVer   int

	// book-keeping params
	requestID  int32
	cmdRunning bool
	cmdStdin   io.Writer
	cmdStdout  io.Reader
	cmdStderr  io.Reader
	replyChan  chan *serverutils.ProbeReply
	success    int64         // toal probe successes
	total      int64         // total number of probes
	latency    time.Duration // cumulative probe latency
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*ProbeConf)
	if !ok {
		return fmt.Errorf("not external probe config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c
	p.replyChan = make(chan *serverutils.ProbeReply)
	p.ipVer = int(p.c.GetIpVersion())

	switch p.c.GetMode() {
	case ProbeConf_ONCE:
		p.mode = "once"
	case ProbeConf_SERVER:
		p.mode = "server"
	default:
		p.l.Errorf("Invalid mode: %s", p.c.GetMode())
	}

	cmdParts := strings.Split(p.c.GetCommand(), " ")
	p.cmdName = cmdParts[0]
	p.cmdArgs = cmdParts[1:len(cmdParts)]

	return nil
}

// substituteLabels replaces occurrences of @label@ with the values from
// abels.  It returns the substituted string and a bool indicating if there
// was a @label@ that did not exist in the labels map.
func substituteLabels(in string, labels map[string]string) (string, bool) {
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

func (p *Probe) startCmdIfNotRunning() error {
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
	cmd := exec.Command(p.cmdName, p.cmdArgs...)
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
			p.l.Infof("Stderr of %s: %s", cmd.Path, scanner.Text())
		}
	}()

	if err = cmd.Start(); err != nil {
		p.l.Errorf("error while starting the cmd: %s %s. Err: %v", cmd.Path, cmd.Args, err)
		return fmt.Errorf("error while starting the cmd: %s %s. Err: %v", cmd.Path, cmd.Args, err)
	}

	done := make(chan struct{})
	// This goroutine waits for the process to terminate and sets cmdRunning to false when that happens.
	go func() {
		err := cmd.Wait()
		close(done)
		p.cmdRunning = false
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				p.l.Errorf("external probe process died with the status: %s. Stderr: %s", exitErr.Error(), string(exitErr.Stderr))
			}
		}
	}()
	go p.readProbeReplies(done)
	p.cmdRunning = true
	return nil
}

func (p *Probe) readProbeReplies(done chan struct{}) {
	// Start a background goroutine to read probe replies from the probe server
	// process's stdout and put them on the probe's replyChan. Note that replyChan
	// is a one element channel. Idea is that we won't need buffering other than the
	// one provided by Unix pipes.
	for {
		select {
		case <-done:
			return
		default:
			rep, err := serverutils.ReadProbeReply(bufio.NewReader(p.cmdStdout))
			if err != nil {
				p.l.Error(err)
			}
			p.replyChan <- rep
		}
	}

}

func (p *Probe) defaultMetrics(target string) *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("success", metrics.NewInt(p.success)).
		AddMetric("total", metrics.NewInt(p.total)).
		AddMetric("latency", metrics.NewFloat(p.latency.Seconds()/p.opts.LatencyUnit.Seconds())).
		AddLabel("ptype", "external").
		AddLabel("probe", p.name).
		AddLabel("dst", target)
}

func (p *Probe) payloadToMetrics(target, payload string) *metrics.EventMetrics {
	em := metrics.NewEventMetrics(time.Now()).
		AddLabel("ptype", "external").
		AddLabel("probe", p.name).
		AddLabel("dst", target)

	switch p.c.GetOutputMetricsKind() {
	case ProbeConf_CUMULATIVE:
		em.Kind = metrics.CUMULATIVE
	case ProbeConf_GAUGE:
		em.Kind = metrics.GAUGE
	case ProbeConf_UNDEFINED:
		if p.c.GetMode() == ProbeConf_ONCE {
			em.Kind = metrics.GAUGE
		} else {
			em.Kind = metrics.CUMULATIVE
		}
	}

	// Convert payload variables into metrics. Variables are specified in
	// the following format:
	// var1 value1
	// var2 value2
	for _, line := range strings.Split(payload, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		varKV := strings.Fields(line)
		if len(varKV) != 2 {
			p.l.Warningf("Wrong var key-value format: %s", line)
			continue
		}
		metricName := varKV[0]
		switch metricName {
		case "success", "total", "latency":
			p.l.Warningf("Metric name (%s) in the output conflicts with standard metrics: (success,total,latency). Ignoring.", metricName)
			continue
		}
		f, err := strconv.ParseFloat(varKV[1], 64)
		if err != nil {
			p.l.Warningf("Only float values are supported: %s", varKV[1])
			continue
		}
		em.AddMetric(metricName, metrics.NewFloat(f))
	}
	// Labels are specified in the probe config.
	if p.c.GetOutputMetricsLabels() != "" {
		for _, label := range strings.Split(p.c.GetOutputMetricsLabels(), ",") {
			labelKV := strings.Split(label, "=")
			if len(labelKV) != 2 {
				p.l.Warningf("Wrong label format: %s", labelKV)
				continue
			}
			em.AddLabel(labelKV[0], labelKV[1])
		}
	}
	return em
}

// replyForProbe looks for a reply on the replyChan, in a context bounded manner.
func (p *Probe) replyForProbe(ctx context.Context, requestID int32) (*serverutils.ProbeReply, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case rep := <-p.replyChan:
			if rep.GetRequestId() != requestID {
				// Not our reply, could be from the last timedout probe. (it shouldn't happen if probe server
				// is using the standard package).
				p.l.Warningf("Got a reply that doesn't match with the request: Request id as per request: %d, as per reply:%s. Ignoring.", requestID, rep.GetRequestId())
				continue
			}
			return rep, nil
		}
	}
}

func (p *Probe) sendRequest(requestID int32, target string) error {
	labels := map[string]string{
		"probe":  p.name,
		"target": target,
	}
	addr, err := p.opts.Targets.Resolve(target, p.ipVer)
	if err != nil {
		p.l.Warningf("Targets.Resolve(%v, %v) failed: %v ", target, p.ipVer, err)
	} else if !addr.IsUnspecified() {
		labels["address"] = addr.String()
	}

	req := &serverutils.ProbeRequest{
		RequestId: proto.Int32(requestID),
		TimeLimit: proto.Int32(int32(p.opts.Timeout / time.Millisecond)),
		Options:   []*serverutils.ProbeRequest_Option{},
	}
	for _, opt := range p.c.GetOptions() {
		value, found := substituteLabels(opt.GetValue(), labels)
		if !found {
			p.l.Warningf("Missing substitution in option %q", value)
		}
		req.Options = append(req.Options, &serverutils.ProbeRequest_Option{
			Name:  opt.Name,
			Value: proto.String(value),
		})
	}
	return serverutils.WriteMessage(req, p.cmdStdin)
}

func (p *Probe) runServerProbe(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	for _, target := range p.opts.Targets.List() {
		// Run probe for the target. Note that we always return probe
		// results regardless of the values of rep and err.
		rep, err := p.runServerProbeForTarget(ctx, target)
		if err != nil {
			p.l.Error(err)
		}
		em := p.defaultMetrics(target)
		p.l.Info(em.String())
		dataChan <- em

		// If we got a non-nil probe reply and probe is configured to use
		// the reply payload as metrics.
		if rep != nil && p.c.GetOutputAsMetrics() {
			em = p.payloadToMetrics(target, rep.GetPayload())
			p.l.Info(em.String())
			dataChan <- em
		}
	}
}

// runServerProbeForTarget runs a "server" probe for a single target and
// returns the ProbeReply and error if any.
func (p *Probe) runServerProbeForTarget(ctx context.Context, target string) (*serverutils.ProbeReply, error) {
	p.total++
	startTime := time.Now()
	if err := p.startCmdIfNotRunning(); err != nil {
		return nil, err
	}
	p.requestID++
	id := p.requestID

	ctxTimeout, cancelFunc := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancelFunc()

	// TODO: We should use context for sending request as well.
	p.l.Debugf("Sending a probe request to the external probe server for target %v", target)
	if err := p.sendRequest(id, target); err != nil {
		return nil, fmt.Errorf("Error sending request to probe server: %v", err)
	}

	rep, err := p.replyForProbe(ctxTimeout, id)
	if err != nil {
		return nil, fmt.Errorf("Error reading reply from probe server: %v", err)
	}
	if rep.GetErrorMessage() != "" {
		return rep, fmt.Errorf("Probe failed with error message: %s", rep.GetErrorMessage())
	}
	p.success++
	p.latency += time.Since(startTime)
	return rep, nil
}

func (p *Probe) runOnceProbe(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	labels := map[string]string{
		"probe": p.name,
	}
	for _, target := range p.opts.Targets.List() {
		labels["target"] = target
		addr, err := p.opts.Targets.Resolve(target, p.ipVer)
		if err != nil {
			p.l.Warningf("Targets.Resolve(%v, %v) failed: %v ", target, p.ipVer, err)
		} else if !addr.IsUnspecified() {
			labels["address"] = addr.String()
		}
		args := make([]string, len(p.cmdArgs))
		for i, arg := range p.cmdArgs {
			res, found := substituteLabels(arg, labels)
			if !found {
				p.l.Warningf("Substitution not found in %q", arg)
			}
			args[i] = res
		}

		p.l.Infof("Running external command: %s %s", p.cmdName, strings.Join(args, " "))
		p.total++
		startTime := time.Now()
		ctxTimeout, cancelFunc := context.WithTimeout(ctx, p.opts.Timeout)
		defer cancelFunc()
		b, err := exec.CommandContext(ctxTimeout, p.cmdName, args...).Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				p.l.Errorf("external probe process died with the status: %s. Stderr: %s", exitErr.Error(), exitErr.Stderr)
			} else {
				p.l.Errorf("Error executing the external program. Err: %v", err)
			}
		} else {
			p.success++
			p.latency += time.Since(startTime)
		}

		em := p.defaultMetrics(target)
		p.l.Info(em.String())
		dataChan <- em

		if p.c.GetOutputAsMetrics() {
			em = p.payloadToMetrics(target, string(b))
			p.l.Info(em.String())
			dataChan <- em
		}
	}
}

func (p *Probe) runProbe(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	if p.mode == "server" {
		p.runServerProbe(ctx, dataChan)
		return
	}
	p.runOnceProbe(ctx, dataChan)
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	for _ = range time.Tick(p.opts.Interval) {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}

		p.runProbe(ctx, dataChan)
	}
}
