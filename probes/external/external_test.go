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

package external

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/external/proto"
	serverpb "github.com/google/cloudprober/probes/external/proto"
	"github.com/google/cloudprober/probes/external/serverutils"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets"
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
func runAndVerifyProbe(t *testing.T, p *Probe, action string, tgts []string, total, success map[string]int64) error {
	setProbeOptions(p, "action", action)
	p.opts.Targets = targets.StaticTargets(strings.Join(tgts, ","))

	dataChan := make(chan *metrics.EventMetrics, 10)
	p.runProbe(context.Background(), dataChan)

	for _, tgt := range p.opts.Targets.List() {
		if p.total[tgt] != total[tgt] {
			t.Errorf("p.total[%s]=%d, Want: %d", tgt, p.total[tgt], total[tgt])
		}
		if p.success[tgt] != success[tgt] {
			t.Errorf("p.success[%s]=%d, Want: %d", tgt, p.success[tgt], success[tgt])
		}
	}
	return nil
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

	p := &Probe{}
	p.Init("testProbe", &options.Options{
		ProbeConf: &configpb.ProbeConf{
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
		},
		Timeout: 1 * time.Second,
	})
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

func TestProbeServer(t *testing.T) {
	p, _, doneChan := testProbeServerSetup(t, nil)
	defer close(doneChan)

	total, success := make(map[string]int64), make(map[string]int64)

	// No payload
	tgts := []string{"target1", "target2"}
	for _, tgt := range tgts {
		total[tgt]++
		success[tgt]++
	}
	runAndVerifyProbe(t, p, "nopayload", tgts, total, success)

	// Payload
	tgts = []string{"target3"}
	for _, tgt := range tgts {
		total[tgt]++
		success[tgt]++
	}
	runAndVerifyProbe(t, p, "payload", tgts, total, success)

	// Payload with error
	tgts = []string{"target2", "target3"}
	for _, tgt := range tgts {
		total[tgt]++
	}
	runAndVerifyProbe(t, p, "payload_with_error", tgts, total, success)

	// Timeout
	tgts = []string{"target1", "target2", "target3"}
	for _, tgt := range tgts {
		total[tgt]++
	}
	// Reduce probe timeout to make this test pass quicker.
	p.opts.Timeout = time.Second
	runAndVerifyProbe(t, p, "timeout", tgts, total, success)
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
	runAndVerifyProbe(t, p, "pipe_server_close", tgts, total, success)
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
	runAndVerifyProbe(t, p, "pipe_local_close", tgts, total, success)
	readError := <-readErrorCh
	if readError == nil {
		t.Error("Didn't get error in reading pipe")
	}
	if _, ok := readError.(*os.PathError); !ok {
		t.Errorf("Didn't get correct error in reading pipe. Got: %T, wanted: *os.PathError", readError)
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
		},
		Targets: targets.StaticTargets("localhost"),
	})
	var buf bytes.Buffer
	p.cmdStdin = &buf

	requestID := int32(1234)
	target := "localhost"

	err := p.sendRequest(requestID, target)
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

func TestLatencyForTarget(t *testing.T) {
	p := &Probe{}
	err := p.Init("testprobe", &options.Options{
		ProbeConf: &configpb.ProbeConf{},
		Targets:   targets.StaticTargets("2.2.2.2"),
	})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}

	latVal := p.latencyForTarget("2.2.2.2")
	if _, ok := latVal.(*metrics.Float); !ok {
		t.Errorf("latency value type is not metrics.Float: %v", latVal)
	}

	// Test with latency distribution option set.
	p.opts.LatencyDist = metrics.NewDistribution([]float64{0.1, 0.2, 0.5})
	latVal = p.latencyForTarget("3.3.3.3")
	if _, ok := latVal.(*metrics.Distribution); !ok {
		t.Errorf("latency value type is not metrics.Distribution: %v", latVal)
	}
}
