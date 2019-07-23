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
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/http/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets"
)

// The Transport is mocked instead of the Client because Client is not an
// interface, but RoundTripper (which Transport implements) is.
type testTransport struct{}

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
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()

	return &http.Response{
		Body: &testReadCloser{
			b: bytes.NewBuffer(b),
		},
	}, nil
}

func (tt *testTransport) CancelRequest(req *http.Request) {}

func testProbe(opts *options.Options) ([]*result, error) {
	p := &Probe{}
	err := p.Init("http_test", opts)
	if err != nil {
		return nil, err
	}
	p.client.Transport = newTestTransport()

	p.runProbe(context.Background())

	var results []*result
	for _, target := range p.targets {
		results = append(results, p.results[target])
	}
	return results, nil
}

func TestRun(t *testing.T) {
	methods := []configpb.ProbeConf_Method{
		configpb.ProbeConf_GET,
		configpb.ProbeConf_POST,
		configpb.ProbeConf_PUT,
		configpb.ProbeConf_HEAD,
		configpb.ProbeConf_DELETE,
		configpb.ProbeConf_PATCH,
		configpb.ProbeConf_OPTIONS,
		100, // Should default to configpb.ProbeConf_GET
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
		{&configpb.ProbeConf{Method: &methods[0]}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: &methods[1]}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: &methods[1], Body: &testBody}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: &methods[2]}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: &methods[2], Body: &testBody}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: &methods[3]}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: &methods[4]}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: &methods[5]}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: &methods[6]}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: &methods[7]}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Headers: []*configpb.ProbeConf_Header{{Name: &testHeaderName, Value: &testHeaderValue}}}, "total: 1, success: 1"},
	}

	for _, test := range tests {
		opts := &options.Options{
			Targets:   targets.StaticTargets("test.com"),
			Interval:  2 * time.Second,
			Timeout:   time.Second,
			ProbeConf: test.input,
		}
		results, err := testProbe(opts)
		if err != nil {
			if fmt.Sprintf("error: '%s'", err.Error()) != test.want {
				t.Errorf("Unexpected initialization error: %v", err)
			}
		} else {
			for _, result := range results {
				got := fmt.Sprintf("total: %d, success: %d", result.total, result.success)
				if got != test.want {
					t.Errorf("Mismatch got '%s', want '%s'", got, test.want)
				}
			}
		}
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

	// Probe 1st run
	p.runProbe(context.Background())
	got := p.results[testTarget].respBodies.String()
	if got != expected {
		t.Errorf("response map: got=%s, expected=%s", got, expected)
	}

	// Probe 2nd run (we should get the same request body).
	p.runProbe(context.Background())
	expectedMap.IncKey(testBody)
	expected = expectedMap.String()
	got = p.results[testTarget].respBodies.String()
	if got != expected {
		t.Errorf("response map: got=%s, expected=%s", got, expected)
	}
}
