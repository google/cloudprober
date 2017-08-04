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

package http

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/targets"
)

// The Transport is mocked instead of the Client because Client is not an
// interface, but RoundTripper (which Transport implements) is.
type testTransport struct{}

func newTestTransport() *testTransport {
	return &testTransport{}
}

// This mocks the Body of an http.Response.
type testReadCloser struct{}

func (trc *testReadCloser) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}
func (trc *testReadCloser) Close() error {
	return nil
}

func (tt *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{Body: &testReadCloser{}}, nil
}
func (tt *testTransport) CancelRequest(req *http.Request) {}

func TestRun(t *testing.T) {
	c := &ProbeConf{
		RequestsPerProbe:        proto.Int32(1),
		StatsExportIntervalMsec: proto.Int32(1000),
	}

	p := &Probe{}
	tgts := targets.StaticTargets("test.com")
	p.Init("http_test", tgts, 2*time.Second, time.Second, nil, c)
	p.client.Transport = newTestTransport()

	resultsChan := make(chan probeutils.ProbeResult, len(p.targets))
	p.runProbe(resultsChan)

	// Strings that should be in all targets' output.
	reqStrs := map[string]int64{
		"sent": 1,
		"rcvd": 1,
	}

	// The resultsChan output iterates through p.targets in the same order.
	for _, target := range p.targets {
		r := <-resultsChan
		result := r.(probeRunResult)
		if result.sent.Int64() != reqStrs["sent"] || result.rcvd.Int64() != reqStrs["rcvd"] {
			t.Errorf("Mismatch got (sent, rcvd) = (%d, %d), want (%d, %d)", result.sent.Int64(), result.rcvd.Int64(), reqStrs["sent"], reqStrs["rcvd"])
		}
		if result.Target() != target {
			t.Errorf("Unexpected target in probe result. Got: %s, Expected: %s", result.Target(), target)
		}
	}
}
