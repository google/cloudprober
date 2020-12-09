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

package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/metrics/testutils"
	configpb "github.com/google/cloudprober/probes/http/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/targets/endpoint"
)

// The Transport is mocked instead of the Client because Client is not an
// interface, but RoundTripper (which Transport implements) is.
type testTransport struct {
	noBody                   io.ReadCloser
	lastProcessedRequestBody []byte
}

func newTestTransport() *testTransport {
	return &testTransport{}
}

// This mocks the Body of an http.Response.
type testReadCloser struct {
	b *bytes.Buffer
}

func (trc *testReadCloser) Read(p []byte) (n int, err error) {
	return trc.b.Read(p)
}

func (trc *testReadCloser) Close() error {
	return nil
}

func (tt *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "fail-test.com" {
		return nil, errors.New("failing for fail-target.com")
	}

	if req.Body == nil {
		return &http.Response{Body: http.NoBody}, nil
	}

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()
	tt.lastProcessedRequestBody = b

	return &http.Response{
		Body: &testReadCloser{
			b: bytes.NewBuffer(b),
		},
	}, nil
}

func (tt *testTransport) CancelRequest(req *http.Request) {}

func testProbe(opts *options.Options) (*probeResult, error) {
	p := &Probe{}
	err := p.Init("http_test", opts)
	if err != nil {
		return nil, err
	}
	p.client.Transport = newTestTransport()

	target := endpoint.Endpoint{Name: "test.com"}
	result := p.newResult()
	req := p.httpRequestForTarget(target, nil)
	p.runProbe(context.Background(), target, req, result)

	return result, nil
}

func TestProbeVariousMethods(t *testing.T) {
	mpb := func(s string) *configpb.ProbeConf_Method {
		return configpb.ProbeConf_Method(configpb.ProbeConf_Method_value[s]).Enum()
	}

	testBody := "Test HTTP Body"
	testHeaderName, testHeaderValue := "Content-Type", "application/json"

	var tests = []struct {
		input *configpb.ProbeConf
		want  string
	}{
		{&configpb.ProbeConf{}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Protocol: configpb.ProbeConf_HTTPS.Enum()}, "total: 1, success: 1"},
		{&configpb.ProbeConf{RequestsPerProbe: proto.Int32(1)}, "total: 1, success: 1"},
		{&configpb.ProbeConf{RequestsPerProbe: proto.Int32(4)}, "total: 4, success: 4"},
		{&configpb.ProbeConf{Method: mpb("GET")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("POST")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("POST"), Body: &testBody}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("PUT")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("PUT"), Body: &testBody}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("HEAD")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("DELETE")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("PATCH")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("OPTIONS")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Headers: []*configpb.ProbeConf_Header{{Name: &testHeaderName, Value: &testHeaderValue}}}, "total: 1, success: 1"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test_case(%d)_config(%v)", i, test.input), func(t *testing.T) {
			opts := &options.Options{
				Targets:   targets.StaticTargets("test.com"),
				Interval:  2 * time.Second,
				Timeout:   time.Second,
				ProbeConf: test.input,
			}

			result, err := testProbe(opts)
			if err != nil {
				if fmt.Sprintf("error: '%s'", err.Error()) != test.want {
					t.Errorf("Unexpected initialization error: %v", err)
				}
				return
			}

			got := fmt.Sprintf("total: %d, success: %d", result.total, result.success)
			if got != test.want {
				t.Errorf("Mismatch got '%s', want '%s'", got, test.want)
			}
		})
	}
}

func TestProbeWithBody(t *testing.T) {
	testBody := "TestHTTPBody"
	testTarget := "test.com"
	// Build the expected response code map
	expectedMap := metrics.NewMap("resp", metrics.NewInt(0))
	expectedMap.IncKey(testBody)
	expected := expectedMap.String()

	p := &Probe{}
	err := p.Init("http_test", &options.Options{
		Targets:  targets.StaticTargets(testTarget),
		Interval: 2 * time.Second,
		ProbeConf: &configpb.ProbeConf{
			Body:                    &testBody,
			ExportResponseAsMetrics: proto.Bool(true),
		},
	})

	if err != nil {
		t.Errorf("Error while initializing probe: %v", err)
	}
	p.client.Transport = newTestTransport()
	target := endpoint.Endpoint{Name: testTarget}

	// Probe 1st run
	result := p.newResult()
	req := p.httpRequestForTarget(target, nil)
	p.runProbe(context.Background(), target, req, result)
	got := result.respBodies.String()
	if got != expected {
		t.Errorf("response map: got=%s, expected=%s", got, expected)
	}

	// Probe 2nd run (we should get the same request body).
	p.runProbe(context.Background(), target, req, result)
	expectedMap.IncKey(testBody)
	expected = expectedMap.String()
	got = result.respBodies.String()
	if got != expected {
		t.Errorf("response map: got=%s, expected=%s", got, expected)
	}
}

func TestProbeWithLargeBody(t *testing.T) {
	for _, size := range []int{largeBodyThreshold - 1, largeBodyThreshold, largeBodyThreshold + 1, largeBodyThreshold * 2} {
		t.Run(fmt.Sprintf("size:%d", size), func(t *testing.T) {
			testProbeWithLargeBody(t, size)
		})
	}
}

func testProbeWithLargeBody(t *testing.T, bodySize int) {
	testBody := strings.Repeat("a", bodySize)
	testTarget := "test-large-body.com"

	p := &Probe{}
	err := p.Init("http_test", &options.Options{
		Targets:  targets.StaticTargets(testTarget),
		Interval: 2 * time.Second,
		ProbeConf: &configpb.ProbeConf{
			Body: &testBody,
			// Can't use ExportResponseAsMetrics for large bodies,
			// since maxResponseSizeForMetrics is small
			ExportResponseAsMetrics: proto.Bool(false),
		},
	})

	if err != nil {
		t.Errorf("Error while initializing probe: %v", err)
	}
	testTransport := newTestTransport()
	p.client.Transport = testTransport
	target := endpoint.Endpoint{Name: testTarget}

	// Probe 1st run
	result := p.newResult()
	req := p.httpRequestForTarget(target, nil)
	p.runProbe(context.Background(), target, req, result)

	got := string(testTransport.lastProcessedRequestBody)
	if got != testBody {
		t.Errorf("response body length: got=%d, expected=%d", len(got), len(testBody))
	}

	// Probe 2nd run (we should get the same request body).
	p.runProbe(context.Background(), target, req, result)
	got = string(testTransport.lastProcessedRequestBody)
	if got != testBody {
		t.Errorf("response body length: got=%d, expected=%d", len(got), len(testBody))
	}
}

func TestMultipleTargetsMultipleRequests(t *testing.T) {
	testTargets := []string{"test.com", "fail-test.com"}
	reqPerProbe := int64(3)
	opts := &options.Options{
		Targets:             targets.StaticTargets(strings.Join(testTargets, ",")),
		Interval:            10 * time.Millisecond,
		StatsExportInterval: 20 * time.Millisecond,
		ProbeConf:           &configpb.ProbeConf{RequestsPerProbe: proto.Int32(int32(reqPerProbe))},
		LogMetrics:          func(_ *metrics.EventMetrics) {},
	}

	p := &Probe{}
	err := p.Init("http_test", opts)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	p.client.Transport = newTestTransport()

	ctx, cancelF := context.WithCancel(context.Background())
	dataChan := make(chan *metrics.EventMetrics, 100)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Start(ctx, dataChan)
	}()

	wantSuccess := map[string]int64{
		"test.com":      2 * reqPerProbe,
		"fail-test.com": 0, // Test transport is configured to fail this.
	}
	time.Sleep(50 * time.Millisecond)

	// We should receive at least 4 eventmetrics: 2 probe cycle x 2 targets.
	ems, err := testutils.MetricsFromChannel(dataChan, 4, time.Second)
	if err != nil {
		t.Errorf("Error getting eventmetrics from data channel: %v", err)
	}

	// Following verifies that we are able to cleanly stop the probe.
	cancelF()
	wg.Wait()

	dataMap := testutils.MetricsMap(ems)
	for tgt, wantSuccessVal := range wantSuccess {
		successVals := dataMap["success"][tgt]
		if len(successVals) < 2 {
			t.Errorf("Success metric for %s: %v (less than 2)", tgt, successVals)
		}
		latestVal := successVals[1].Metric("success").(*metrics.Int).Int64()
		if latestVal < wantSuccessVal {
			t.Errorf("Got success value for target (%s): %d, want: %d", tgt, latestVal, wantSuccessVal)
		}
	}
}

func compareNumberOfMetrics(t *testing.T, ems []*metrics.EventMetrics, targets [2]string, wantCloseRange bool) {
	t.Helper()

	m := testutils.MetricsMap(ems)["success"]
	num1 := len(m[targets[0]])
	num2 := len(m[targets[1]])

	diff := num1 - num2
	threshold := num1 / 2
	notCloseRange := diff < -(threshold) || diff > threshold

	if notCloseRange && wantCloseRange {
		t.Errorf("Number of metrics for two targets are not within a close range (%d, %d)", num1, num2)
	}
	if !notCloseRange && !wantCloseRange {
		t.Errorf("Number of metrics for two targets are within a close range (%d, %d)", num1, num2)
	}
}

func TestUpdateTargetsAndStartProbes(t *testing.T) {
	testTargets := [2]string{"test1.com", "test2.com"}
	reqPerProbe := int64(3)
	opts := &options.Options{
		Targets:             targets.StaticTargets(fmt.Sprintf("%s,%s", testTargets[0], testTargets[1])),
		Interval:            10 * time.Millisecond,
		StatsExportInterval: 20 * time.Millisecond,
		ProbeConf:           &configpb.ProbeConf{RequestsPerProbe: proto.Int32(int32(reqPerProbe))},
		LogMetrics:          func(_ *metrics.EventMetrics) {},
	}
	p := &Probe{}
	p.Init("http_test", opts)
	p.client.Transport = newTestTransport()

	dataChan := make(chan *metrics.EventMetrics, 100)

	ctx, cancelF := context.WithCancel(context.Background())
	p.updateTargetsAndStartProbes(ctx, dataChan)
	if len(p.cancelFuncs) != 2 {
		t.Errorf("len(p.cancelFunc)=%d, want=2", len(p.cancelFuncs))
	}
	ems, _ := testutils.MetricsFromChannel(dataChan, 100, time.Second)
	compareNumberOfMetrics(t, ems, testTargets, true)

	// Updates targets to just one target. This should cause one probe loop to
	// exit. We should get only one data stream after that.
	opts.Targets = targets.StaticTargets(testTargets[0])
	p.updateTargetsAndStartProbes(ctx, dataChan)
	if len(p.cancelFuncs) != 1 {
		t.Errorf("len(p.cancelFunc)=%d, want=1", len(p.cancelFuncs))
	}
	ems, _ = testutils.MetricsFromChannel(dataChan, 100, time.Second)
	compareNumberOfMetrics(t, ems, testTargets, false)

	cancelF()
	p.wait()
}
