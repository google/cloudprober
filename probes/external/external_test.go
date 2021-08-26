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

package external

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	payloadconfigpb "github.com/google/cloudprober/metrics/payload/proto"
	"github.com/google/cloudprober/metrics/testutils"
	configpb "github.com/google/cloudprober/probes/external/proto"
	serverpb "github.com/google/cloudprober/probes/external/proto"
	"github.com/google/cloudprober/probes/external/serverutils"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/targets/endpoint"
)

func isDone(doneChan chan struct{}) bool {
	// If we are done, return immediately.
	select {
	case <-doneChan:
		return true
	default:
	}
	return false
}

// startProbeServer starts a test probe server to work with the TestProbeServer
// test below.
func startProbeServer(t *testing.T, testPayload string, r io.Reader, w io.WriteCloser, doneChan chan struct{}) {
	rd := bufio.NewReader(r)
	for {
		if isDone(doneChan) {
			return
		}

		req, err := serverutils.ReadProbeRequest(rd)
		if err != nil {
			// Normal failure because we are finished.
			if isDone(doneChan) {
				return
			}
			t.Errorf("Error reading probe request. Err: %v", err)
			return
		}
		var action, target string
		opts := req.GetOptions()
		for _, opt := range opts {
			if opt.GetName() == "action" {
				action = opt.GetValue()
				continue
			}
			if opt.GetName() == "target" {
				target = opt.GetValue()
				continue
			}
		}
		id := req.GetRequestId()

		actionToResponse := map[string]*serverpb.ProbeReply{
			"nopayload": &serverpb.ProbeReply{RequestId: proto.Int32(id)},
			"payload": &serverpb.ProbeReply{
				RequestId: proto.Int32(id),
				Payload:   proto.String(testPayload),
			},
			"payload_with_error": &serverpb.ProbeReply{
				RequestId:    proto.Int32(id),
				Payload:      proto.String(testPayload),
				ErrorMessage: proto.String("error"),
			},
		}
		t.Logf("Request id: %d, action: %s, target: %s", id, action, target)
		if action == "pipe_server_close" {
			w.Close()
			return
		}
		if res, ok := actionToResponse[action]; ok {
			serverutils.WriteMessage(res, w)
		}
	}
}

func setProbeOptions(p *Probe, name, value string) {
	for _, opt := range p.c.Options {
		if opt.GetName() == name {
			opt.Value = proto.String(value)
			break
		}
	}
}

// runAndVerifyServerProbe executes a server probe and verifies the replies
// received.
func runAndVerifyServerProbe(t *testing.T, p *Probe, action string, tgts []string, total, success map[string]int64, numEventMetrics int) {
	setProbeOptions(p, "action", action)

	runAndVerifyProbe(t, p, tgts, total, success)

	// Verify that we got all the expected EventMetrics
	ems, err := testutils.MetricsFromChannel(p.dataChan, numEventMetrics, 1*time.Second)
	if err != nil {
		t.Error(err)
	}
	metricsMap := testutils.MetricsMap(ems)

	// Convenient wrapper to get the last value from a series.
	lastValue := func(s []*metrics.EventMetrics, metricName string) int64 {
		return s[len(s)-1].Metric(metricName).(metrics.NumValue).Int64()
	}

	for _, tgt := range tgts {
		vals := make(map[string]int64)
		for _, m := range []string{"total", "success"} {
			s := metricsMap[m][tgt]
			if len(s) == 0 {
				t.Errorf("No %s metric for target: %s", m, tgt)
				continue
			}
			vals[m] = lastValue(s, m)
		}
		if vals["success"] != success[tgt] || vals["total"] != total[tgt] {
			t.Errorf("Target(%s) total=%d, success=%d, wanted: total=%d, success=%d, all_metrics=%s", tgt, vals["total"], vals["success"], total[tgt], success[tgt], ems)
		}
	}
}

func runAndVerifyProbe(t *testing.T, p *Probe, tgts []string, total, success map[string]int64) {
	p.opts.Targets = targets.StaticTargets(strings.Join(tgts, ","))
	p.updateTargets()

	p.runProbe(context.Background())

	for _, target := range p.targets {
		tgt := target.Name

		if p.results[tgt].total != total[tgt] {
			t.Errorf("p.total[%s]=%d, Want: %d", tgt, p.results[tgt].total, total[tgt])
		}
		if p.results[tgt].success != success[tgt] {
			t.Errorf("p.success[%s]=%d, Want: %d", tgt, p.results[tgt].success, success[tgt])
		}
	}
}

func createTestProbe(cmd string) *Probe {
	probeConf := &configpb.ProbeConf{
		Options: []*configpb.ProbeConf_Option{
			{
				Name:  proto.String("target"),
				Value: proto.String("@target@"),
			},
			{
				Name:  proto.String("action"),
				Value: proto.String(""),
			},
		},
		Command: &cmd,
	}

	p := &Probe{
		dataChan: make(chan *metrics.EventMetrics, 20),
	}

	p.Init("testProbe", &options.Options{
		ProbeConf:  probeConf,
		Timeout:    1 * time.Second,
		LogMetrics: func(em *metrics.EventMetrics) {},
	})

	return p
}

func testProbeServerSetup(t *testing.T, readErrorCh chan error) (*Probe, string, chan struct{}) {
	// We create two pairs of pipes to establish communication between this prober
	// and the test probe server (defined above).
	// Test probe server input pipe. We writes on w1 and external command reads
	// from r1.
	r1, w1, err := os.Pipe()
	if err != nil {
		t.Errorf("Error creating OS pipe. Err: %v", err)
	}
	// Test probe server output pipe. External command writes on w2 and we read
	// from r2.
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Errorf("Error creating OS pipe. Err: %v", err)
	}

	testPayload := "p90 45\n"
	// Start probe server in a goroutine
	doneChan := make(chan struct{})
	go startProbeServer(t, testPayload, r1, w2, doneChan)

	p := createTestProbe("./testCommand")
	p.cmdRunning = true // don't try to start the probe server
	p.cmdStdin = w1
	p.cmdStdout = r2
	p.mode = "server"

	// Start the goroutine that reads probe replies.
	go func() {
		err := p.readProbeReplies(doneChan)
		if readErrorCh != nil {
			readErrorCh <- err
			close(readErrorCh)
		}
	}()

	return p, testPayload, doneChan
}

func TestProbeServerMode(t *testing.T) {
	p, _, doneChan := testProbeServerSetup(t, nil)
	defer close(doneChan)

	total, success := make(map[string]int64), make(map[string]int64)

	// No payload
	tgts := []string{"target1", "target2"}
	for _, tgt := range tgts {
		total[tgt]++
		success[tgt]++
	}
	t.Run("nopayload", func(t *testing.T) {
		runAndVerifyServerProbe(t, p, "nopayload", tgts, total, success, 2)
	})

	// Payload
	tgts = []string{"target3"}
	for _, tgt := range tgts {
		total[tgt]++
		success[tgt]++
	}
	t.Run("payload", func(t *testing.T) {
		// 2 metrics per target
		runAndVerifyServerProbe(t, p, "payload", tgts, total, success, 1*2)
	})

	// Payload with error
	tgts = []string{"target2", "target3"}
	for _, tgt := range tgts {
		total[tgt]++
	}
	t.Run("payload_with_error", func(t *testing.T) {
		// 2 targets, 2 EMs per target
		runAndVerifyServerProbe(t, p, "payload_with_error", tgts, total, success, 2*2)
	})

	// Timeout
	tgts = []string{"target1", "target2", "target3"}
	for _, tgt := range tgts {
		total[tgt]++
	}

	// Reduce probe timeout to make this test pass quicker.
	p.opts.Timeout = time.Second
	t.Run("timeout", func(t *testing.T) {
		// 3 targets, 1 EM per target
		runAndVerifyServerProbe(t, p, "timeout", tgts, total, success, 3*1)
	})
}

func TestProbeServerRemotePipeClose(t *testing.T) {
	readErrorCh := make(chan error)
	p, _, doneChan := testProbeServerSetup(t, readErrorCh)
	defer close(doneChan)

	total, success := make(map[string]int64), make(map[string]int64)
	// Remote pipe close
	tgts := []string{"target"}
	for _, tgt := range tgts {
		total[tgt]++
	}
	// Reduce probe timeout to make this test pass quicker.
	p.opts.Timeout = time.Second
	runAndVerifyServerProbe(t, p, "pipe_server_close", tgts, total, success, 1)
	readError := <-readErrorCh
	if readError == nil {
		t.Error("Didn't get error in reading pipe")
	}
	if readError != io.EOF {
		t.Errorf("Didn't get correct error in reading pipe. Got: %v, wanted: %v", readError, io.EOF)
	}
}

func TestProbeServerLocalPipeClose(t *testing.T) {
	readErrorCh := make(chan error)
	p, _, doneChan := testProbeServerSetup(t, readErrorCh)
	defer close(doneChan)

	total, success := make(map[string]int64), make(map[string]int64)
	// Local pipe close
	tgts := []string{"target"}
	for _, tgt := range tgts {
		total[tgt]++
	}
	// Reduce probe timeout to make this test pass quicker.
	p.opts.Timeout = time.Second
	p.cmdStdout.(*os.File).Close()
	runAndVerifyServerProbe(t, p, "pipe_local_close", tgts, total, success, 1)
	readError := <-readErrorCh
	if readError == nil {
		t.Error("Didn't get error in reading pipe")
	}
	if _, ok := readError.(*os.PathError); !ok {
		t.Errorf("Didn't get correct error in reading pipe. Got: %T, wanted: *os.PathError", readError)
	}
}

func TestProbeOnceMode(t *testing.T) {
	testCmd := "/test/cmd --arg1 --arg2"

	p := createTestProbe(testCmd)
	p.mode = "once"
	tgts := []string{"target1", "target2"}

	oldRunCommand := runCommand
	defer func() { runCommand = oldRunCommand }()

	// Set runCommand to a function that runs successfully and returns a pyload.
	runCommand = func(ctx context.Context, cmd string, cmdArgs []string) ([]byte, error) {
		var resp []string
		resp = append(resp, fmt.Sprintf("cmd \"%s\"", cmd))
		resp = append(resp, fmt.Sprintf("num-args %d", len(cmdArgs)))
		return []byte(strings.Join(resp, "\n")), nil
	}

	total, success := make(map[string]int64), make(map[string]int64)

	for _, tgt := range tgts {
		total[tgt]++
		success[tgt]++
	}

	runAndVerifyProbe(t, p, tgts, total, success)

	// Try with failing command now
	runCommand = func(ctx context.Context, cmd string, cmdArgs []string) ([]byte, error) {
		return nil, fmt.Errorf("error executing %s", cmd)
	}

	for _, tgt := range tgts {
		total[tgt]++
	}
	runAndVerifyProbe(t, p, tgts, total, success)

	// Total numbder of event metrics:
	// num_of_runs x num_targets x (1 for default metrics + 1 for payload metrics)
	ems, err := testutils.MetricsFromChannel(p.dataChan, 8, time.Second)
	if err != nil {
		t.Error(err)
	}
	metricsMap := testutils.MetricsMap(ems)

	if metricsMap["num-args"] == nil && metricsMap["cmd"] == nil {
		t.Errorf("Didn't get all metrics from the external process output.")
	}

	if metricsMap["total"] == nil && metricsMap["success"] == nil {
		t.Errorf("Didn't get default metrics from the probe run.")
	}

	for _, tgt := range tgts {
		// Verify that default metrics were received for both runs -- success and
		// failure. We don't check for the values here as that's already done by
		// runAndVerifyProbe to an extent.
		for _, m := range []string{"total", "success", "latency"} {
			if len(metricsMap[m][tgt]) != 2 {
				t.Errorf("Wrong number of values for default metric (%s) for target (%s). Got=%d, Expected=2", m, tgt, len(metricsMap[m][tgt]))
			}
		}

		for _, m := range []string{"num-args", "cmd"} {
			if len(metricsMap[m][tgt]) != 1 {
				t.Errorf("Wrong number of values for metric (%s) for target (%s) from the command output. Got=%d, Expected=1", m, tgt, len(metricsMap[m][tgt]))
			}
		}

		tgtNumArgs := metricsMap["num-args"][tgt][0].Metric("num-args").(metrics.NumValue).Int64()
		expectedNumArgs := int64(len(strings.Split(testCmd, " ")) - 1)
		if tgtNumArgs != expectedNumArgs {
			t.Errorf("Wrong metric value for target (%s) from the command output. Got=%d, Expected=%d", tgt, tgtNumArgs, expectedNumArgs)
		}

		tgtCmd := metricsMap["cmd"][tgt][0].Metric("cmd").String()
		expectedCmd := fmt.Sprintf("\"%s\"", strings.Split(testCmd, " ")[0])
		if tgtCmd != expectedCmd {
			t.Errorf("Wrong metric value for target (%s) from the command output. got=%s, expected=%s", tgt, tgtCmd, expectedCmd)
		}
	}
}

func TestUpdateLabelKeys(t *testing.T) {
	c := &configpb.ProbeConf{
		Options: []*configpb.ProbeConf_Option{
			{
				Name:  proto.String("target"),
				Value: proto.String("@target@"),
			},
			{
				Name:  proto.String("probe"),
				Value: proto.String("@probe@"),
			},
		},
	}
	p := &Probe{
		name:    "probeP",
		c:       c,
		cmdArgs: []string{"--server", "@target.label.fqdn@:@port@"},
	}

	p.updateLabelKeys()

	expected := map[string]bool{
		"target":            true,
		"port":              true,
		"probe":             true,
		"target.label.fqdn": true,
	}

	if !reflect.DeepEqual(p.labelKeys, expected) {
		t.Errorf("p.labelKeys got: %v, want: %v", p.labelKeys, expected)
	}

	gotLabels := p.labels(endpoint.Endpoint{
		Name: "targetA",
		Port: 8080,
		Labels: map[string]string{
			"fqdn": "targetA.svc.local",
		},
	})
	wantLabels := map[string]string{
		"target.label.fqdn": "targetA.svc.local",
		"port":              "8080",
		"probe":             "probeP",
		"target":            "targetA",
	}
	if !reflect.DeepEqual(gotLabels, wantLabels) {
		t.Errorf("p.labels got: %v, want: %v", gotLabels, wantLabels)
	}
}

func TestSubstituteLabels(t *testing.T) {
	tests := []struct {
		desc   string
		in     string
		labels map[string]string
		want   string
		found  bool
	}{
		{
			desc:  "No replacement",
			in:    "foo bar baz",
			want:  "foo bar baz",
			found: true,
		},
		{
			desc: "Replacement beginning",
			in:   "@foo@ bar baz",
			labels: map[string]string{
				"foo": "h e llo",
			},
			want:  "h e llo bar baz",
			found: true,
		},
		{
			desc: "Replacement middle",
			in:   "beginning @😿@ end",
			labels: map[string]string{
				"😿": "😺",
			},
			want:  "beginning 😺 end",
			found: true,
		},
		{
			desc: "Replacement end",
			in:   "bar baz @foo@",
			labels: map[string]string{
				"foo": "XöX",
				"bar": "nope",
			},
			want:  "bar baz XöX",
			found: true,
		},
		{
			desc: "Replacements",
			in:   "abc@foo@def@foo@ jk",
			labels: map[string]string{
				"def": "nope",
				"foo": "XöX",
			},
			want:  "abcXöXdefXöX jk",
			found: true,
		},
		{
			desc: "Multiple labels",
			in:   "xx @foo@@bar@ yy",
			labels: map[string]string{
				"bar": "_",
				"def": "nope",
				"foo": "XöX",
			},
			want:  "xx XöX_ yy",
			found: true,
		},
		{
			desc: "Not found",
			in:   "A b C @d@ e",
			labels: map[string]string{
				"bar": "_",
				"def": "nope",
				"foo": "XöX",
			},
			want: "A b C @d@ e",
		},
		{
			desc: "@@",
			in:   "hello@@foo",
			labels: map[string]string{
				"bar": "_",
				"def": "nope",
				"foo": "XöX",
			},
			want:  "hello@foo",
			found: true,
		},
		{
			desc: "odd number",
			in:   "hello@foo@bar@xx",
			labels: map[string]string{
				"foo": "yy",
			},
			want:  "helloyybar@xx",
			found: true,
		},
	}

	for _, tc := range tests {
		got, found := substituteLabels(tc.in, tc.labels)
		if tc.found != found {
			t.Errorf("%v: substituteLabels(%q, %q) = _, %v, want %v", tc.desc, tc.in, tc.labels, found, tc.found)
		}
		if tc.want != got {
			t.Errorf("%v: substituteLabels(%q, %q) = %q, _, want %q", tc.desc, tc.in, tc.labels, got, tc.want)
		}
	}
}

// TestSendRequest verifies that sendRequest sends appropriately populated
// ProbeRequest.
func TestSendRequest(t *testing.T) {
	p := &Probe{}
	p.Init("testprobe", &options.Options{
		ProbeConf: &configpb.ProbeConf{
			Options: []*configpb.ProbeConf_Option{
				{
					Name:  proto.String("target"),
					Value: proto.String("@target@"),
				},
			},
			Command: proto.String("./testCommand"),
		},
		Targets: targets.StaticTargets("localhost"),
	})
	var buf bytes.Buffer
	p.cmdStdin = &buf

	requestID := int32(1234)
	target := "localhost"

	err := p.sendRequest(requestID, endpoint.Endpoint{Name: target})
	if err != nil {
		t.Errorf("Failed to sendRequest: %v", err)
	}
	req := new(serverpb.ProbeRequest)
	var length int
	_, err = fmt.Fscanf(&buf, "\nContent-Length: %d\n\n", &length)
	if err != nil {
		t.Errorf("Failed to read header: %v", err)
	}
	err = proto.Unmarshal(buf.Bytes(), req)
	if err != nil {
		t.Fatalf("Failed to Unmarshal probe Request: %v", err)
	}
	if got, want := req.GetRequestId(), requestID; got != requestID {
		t.Errorf("req.GetRequestId() = %q, want %v", got, want)
	}
	opts := req.GetOptions()
	if len(opts) != 1 {
		t.Errorf("req.GetOptions() = %q (%v), want only one item", opts, len(opts))
	}
	if got, want := opts[0].GetName(), "target"; got != want {
		t.Errorf("opts[0].GetName() = %q, want %q", got, want)
	}
	if got, want := opts[0].GetValue(), target; got != target {
		t.Errorf("opts[0].GetValue() = %q, want %q", got, want)
	}
}

func TestUpdateTargets(t *testing.T) {
	p := &Probe{}
	err := p.Init("testprobe", &options.Options{
		ProbeConf: &configpb.ProbeConf{
			Command: proto.String("./testCommand"),
		},
		Targets: targets.StaticTargets("2.2.2.2"),
	})
	if err != nil {
		t.Fatalf("Got error while initializing the probe: %v", err)
	}

	p.updateTargets()
	latVal := p.results["2.2.2.2"].latency
	if _, ok := latVal.(*metrics.Float); !ok {
		t.Errorf("latency value type is not metrics.Float: %v", latVal)
	}

	// Test with latency distribution option set.
	p.opts.LatencyDist = metrics.NewDistribution([]float64{0.1, 0.2, 0.5})
	delete(p.results, "2.2.2.2")
	p.updateTargets()
	latVal = p.results["2.2.2.2"].latency
	if _, ok := latVal.(*metrics.Distribution); !ok {
		t.Errorf("latency value type is not metrics.Distribution: %v", latVal)
	}
}

func verifyProcessedResult(t *testing.T, p *Probe, r *result, success int64, name string, val int64, payloadLabels [][2]string) {
	t.Helper()

	testTarget := "test-target"
	if r.success != success {
		t.Errorf("r.success=%d, expected=%d", r.success, success)
	}

	m, err := testutils.MetricsFromChannel(p.dataChan, 2, time.Second)
	if err != nil {
		t.Fatal(err.Error())
	}

	metricsMap := testutils.MetricsMap(m)

	if metricsMap[name] == nil || len(metricsMap[name][testTarget]) < 1 {
		t.Fatalf("Payload metric %s is missing in %+v", name, metricsMap)
	}

	em := metricsMap[name][testTarget][0]
	gotValue := em.Metric(name).(metrics.NumValue).Int64()
	if gotValue != val {
		t.Errorf("%s=%d, expected=%d", name, gotValue, val)
	}

	expectedLabels := map[string]string{"ptype": "external", "probe": "testprobe", "dst": "test-target"}
	for _, kv := range payloadLabels {
		expectedLabels[kv[0]] = kv[1]
	}
	for key, val := range expectedLabels {
		if em.Label(key) != val {
			t.Errorf("r.payloadMetrics.Label(%s)=%s, expected=%s", key, r.payloadMetrics.Label(key), val)
		}
	}
}

func TestProcessProbeResult(t *testing.T) {
	for _, agg := range []bool{true, false} {

		t.Run(fmt.Sprintf("With aggregation: %v", agg), func(t *testing.T) {

			p := &Probe{}
			opts := options.DefaultOptions()
			opts.ProbeConf = &configpb.ProbeConf{
				OutputMetricsOptions: &payloadconfigpb.OutputMetricsOptions{
					AggregateInCloudprober: proto.Bool(agg),
				},
				Command: proto.String("./testCommand"),
			}
			err := p.Init("testprobe", opts)
			if err != nil {
				t.Fatal(err)
			}

			p.dataChan = make(chan *metrics.EventMetrics, 20)

			r := &result{
				latency: metrics.NewFloat(0),
			}

			payloadMetricName := map[bool]string{
				false: "p-failures{service=serviceA,db=dbA}",
				true:  "p-failures",
			}
			payloadLabels := map[bool][][2]string{
				false: [][2]string{
					[2]string{"service", "serviceA"},
					[2]string{"db", "dbA"},
				},
			}

			// First run
			p.processProbeResult(&probeStatus{
				target:  "test-target",
				success: true,
				latency: time.Millisecond,
				payload: fmt.Sprintf("%s 14", payloadMetricName[agg]),
			}, r)

			verifyProcessedResult(t, p, r, 1, "p-failures", 14, payloadLabels[agg])

			// Second run
			p.processProbeResult(&probeStatus{
				target:  "test-target",
				success: true,
				latency: time.Millisecond,
				payload: fmt.Sprintf("%s 11", payloadMetricName[agg]),
			}, r)

			if agg {
				verifyProcessedResult(t, p, r, 2, "p-failures", 25, payloadLabels[agg])
			} else {
				verifyProcessedResult(t, p, r, 2, "p-failures", 11, payloadLabels[agg])
			}
		})
	}
}

func TestCommandParsing(t *testing.T) {
	p := createTestProbe("./test-command --flag1 one --flag23 \"two three\"")

	wantCmdName := "./test-command"
	if p.cmdName != wantCmdName {
		t.Errorf("Got command name=%s, want command name=%s", p.cmdName, wantCmdName)
	}

	wantArgs := []string{"--flag1", "one", "--flag23", "two three"}
	if !reflect.DeepEqual(p.cmdArgs, wantArgs) {
		t.Errorf("Got command args=%v, want command args=%v", p.cmdArgs, wantArgs)
	}
}

type fakeCommand struct {
	exitCtx  context.Context
	startCtx context.Context
	waitErr  error
}

func (fc *fakeCommand) Wait() error {
	select {
	case <-fc.exitCtx.Done():
	case <-fc.startCtx.Done():
	}
	return fc.waitErr
}

func TestMonitorCommand(t *testing.T) {
	tests := []struct {
		desc       string
		waitErr    error
		finishCmd  bool
		cancelCtx  bool
		wantErr    bool
		wantStderr bool
	}{
		{
			desc:      "Command exit with no error",
			finishCmd: true,
			wantErr:   false,
		},
		{
			desc:      "Cancel context, no error",
			cancelCtx: true,
			wantErr:   false,
		},
		{
			desc:       "command exit with exit error",
			finishCmd:  true,
			waitErr:    &exec.ExitError{Stderr: []byte("exit-error exiting")},
			wantErr:    true,
			wantStderr: true,
		},
		{
			desc:       "command exit with no exit error",
			finishCmd:  true,
			waitErr:    errors.New("some-error"),
			wantErr:    true,
			wantStderr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			exitCtx, exitFunc := context.WithCancel(context.Background())
			startCtx, startCancelFunc := context.WithCancel(context.Background())
			cmd := &fakeCommand{
				exitCtx:  exitCtx,
				startCtx: startCtx,
				waitErr:  test.waitErr,
			}

			p := &Probe{}
			errCh := make(chan error)
			go func() {
				errCh <- p.monitorCommand(startCtx, cmd)
			}()

			if test.finishCmd {
				exitFunc()
			}
			if test.cancelCtx {
				startCancelFunc()
			}

			err := <-errCh
			if (err != nil) != test.wantErr {
				t.Errorf("Got error: %v, want error?= %v", err, test.wantErr)
			}

			if err != nil {
				if test.wantStderr && !strings.Contains(err.Error(), "Stderr") {
					t.Errorf("Want std err: %v, got std err: %v", test.wantStderr, strings.Contains(err.Error(), "Stderr"))
				}
			}

			exitFunc()
			startCancelFunc()
		})
	}
}
