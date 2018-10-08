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
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/external/proto"
	serverpb "github.com/google/cloudprober/probes/external/proto"
	"github.com/google/cloudprober/probes/external/serverutils"
	"github.com/google/cloudprober/probes/options"
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
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name    string
	mode    string
	cmdName string
	cmdArgs []string
	opts    *options.Options
	c       *configpb.ProbeConf
	l       *logger.Logger
	ipVer   int

	// book-keeping params
	labelKeys  map[string]bool // Labels for substitution
	requestID  int32
	cmdRunning bool
	cmdStdin   io.Writer
	cmdStdout  io.ReadCloser
	cmdStderr  io.ReadCloser
	replyChan  chan *serverpb.ProbeReply
	success    map[string]int64         // total probe successes
	total      map[string]int64         // total number of probes
	latency    map[string]time.Duration // cumulative probe latency, in microseconds.

	// EventMetrics created from external probe process output
	defaultPayloadMetrics *metrics.EventMetrics
	payloadMetrics        map[string]*metrics.EventMetrics // Per-target metrics
	payloadMetricsMu      sync.Mutex
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
	p.labelKeys = make(map[string]bool)
	validLabels := []string{"@target@", "@address@", "@probe@"}
	for _, l := range validLabels {
		for _, opt := range p.c.GetOptions() {
			if strings.Contains(opt.GetValue(), l) {
				p.labelKeys[l] = true
			}
		}
		for _, arg := range p.cmdArgs {
			if strings.Contains(arg, l) {
				p.labelKeys[l] = true
			}
		}
	}

	switch p.c.GetMode() {
	case configpb.ProbeConf_ONCE:
		p.mode = "once"
	case configpb.ProbeConf_SERVER:
		p.mode = "server"
	default:
		p.l.Errorf("Invalid mode: %s", p.c.GetMode())
	}

	p.success = make(map[string]int64)
	p.total = make(map[string]int64)
	p.latency = make(map[string]time.Duration)

	p.payloadMetrics = make(map[string]*metrics.EventMetrics)
	return p.initPayloadMetrics()
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

func (p *Probe) defaultMetrics(target string) *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("success", metrics.NewInt(p.success[target])).
		AddMetric("total", metrics.NewInt(p.total[target])).
		AddMetric("latency", metrics.NewFloat(p.latency[target].Seconds()/p.opts.LatencyUnit.Seconds())).
		AddLabel("ptype", "external").
		AddLabel("probe", p.name).
		AddLabel("dst", target)
}

func (p *Probe) labels(target string) map[string]string {
	labels := make(map[string]string)
	if p.labelKeys["@probe@"] {
		labels["probe"] = p.name
	}
	if p.labelKeys["@target@"] {
		labels["target"] = target
	}
	if p.labelKeys["@address@"] {
		addr, err := p.opts.Targets.Resolve(target, p.ipVer)
		if err != nil {
			p.l.Warningf("Targets.Resolve(%v, %v) failed: %v ", target, p.ipVer, err)
		} else if !addr.IsUnspecified() {
			labels["address"] = addr.String()
		}
	}
	return labels
}

func (p *Probe) sendRequest(requestID int32, target string) error {
	req := &serverpb.ProbeRequest{
		RequestId: proto.Int32(requestID),
		TimeLimit: proto.Int32(int32(p.opts.Timeout / time.Millisecond)),
		Options:   []*serverpb.ProbeRequest_Option{},
	}
	for _, opt := range p.c.GetOptions() {
		value, found := substituteLabels(opt.GetValue(), p.labels(target))
		if !found {
			p.l.Warningf("Missing substitution in option %q", value)
		}
		req.Options = append(req.Options, &serverpb.ProbeRequest_Option{
			Name:  opt.Name,
			Value: proto.String(value),
		})
	}

	p.l.Debugf("Sending a probe request %v to the external probe server for target %v", requestID, target)
	return serverutils.WriteMessage(req, p.cmdStdin)
}

type requestInfo struct {
	target    string
	timestamp time.Time
}

func (p *Probe) runServerProbe(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	requests := make(map[int32]requestInfo)
	var requestsMu sync.RWMutex
	doneChan := make(chan struct{})

	if err := p.startCmdIfNotRunning(); err != nil {
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
				if rep.GetErrorMessage() != "" {
					p.l.Errorf("Probe for target %v failed with error message: %s", reqInfo.target, rep.GetErrorMessage())
				} else {
					p.success[reqInfo.target]++
					p.latency[reqInfo.target] += time.Since(reqInfo.timestamp)
				}
				em := p.defaultMetrics(reqInfo.target)
				dataChan <- em

				// If we got a non-nil probe reply and probe is configured to use
				// the reply payload as metrics.
				if rep != nil && p.c.GetOutputAsMetrics() {
					em = p.payloadToMetrics(reqInfo.target, rep.GetPayload())
					dataChan <- em
				}
			}
		}
	}()

	// Send probe requests
	for _, target := range p.opts.Targets.List() {
		p.requestID++
		p.total[target]++
		requestsMu.Lock()
		requests[p.requestID] = requestInfo{
			target:    target,
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

func (p *Probe) runOnceProbe(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	var wg sync.WaitGroup
	for _, target := range p.opts.Targets.List() {
		wg.Add(1)
		go func(target string) {
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
			p.total[target]++
			startTime := time.Now()
			b, err := exec.CommandContext(ctx, p.cmdName, args...).Output()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					p.l.Errorf("external probe process died with the status: %s. Stderr: %s", exitErr.Error(), exitErr.Stderr)
				} else {
					p.l.Errorf("Error executing the external program. Err: %v", err)
				}
			} else {
				p.success[target]++
				p.latency[target] += time.Since(startTime)
			}

			em := p.defaultMetrics(target)
			p.l.Info(em.String())
			dataChan <- em

			if p.c.GetOutputAsMetrics() {
				em = p.payloadToMetrics(target, string(b))
				p.l.Info(em.String())
				dataChan <- em
			}
		}(target)
	}
	wg.Wait()
}

func (p *Probe) runProbe(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	ctxTimeout, cancelFunc := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancelFunc()
	if p.mode == "server" {
		p.runServerProbe(ctxTimeout, dataChan)
		return
	}
	p.runOnceProbe(ctxTimeout, dataChan)
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
