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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	configpb "github.com/google/cloudprober/probes/http/proto"
	"github.com/google/cloudprober/probes/options"
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

func testProbe(opts *options.Options) ([]probeRunResult, error) {
	p := &Probe{}
	err := p.Init("http_test", opts)
	if err != nil {
		return nil, err
	}
	p.client.Transport = newTestTransport()

	resultsChan := make(chan probeutils.ProbeResult, len(p.targets))
	p.runProbe(resultsChan)

	results := make([]probeRunResult, len(p.targets))
	// The resultsChan output iterates through p.targets in the same order.
	for i := range p.targets {
		r := <-resultsChan
		results[i] = r.(probeRunResult)
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
		{&configpb.ProbeConf{RequestsPerProbe: proto.Int32(1), StatsExportIntervalMsec: proto.Int32(1000)}, "total: 1, success: 1"},
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
			for i, result := range results {
				got := fmt.Sprintf("total: %d, success: %d", result.total.Int64(), result.success.Int64())
				if got != test.want {
					t.Errorf("Mismatch got '%s', want '%s'", got, test.want)
				}
				if result.Target() != opts.Targets.List()[i] {
					t.Errorf("Unexpected target in probe result. Got: %s, Expected: %s", result.Target(), opts.Targets.List()[i])
				}
			}
		}
	}
}

type intf struct {
	addrs []net.Addr
}

func (i *intf) Addrs() ([]net.Addr, error) {
	return i.addrs, nil
}

func mockInterfaceByName(iname string, addrs []string) {
	ips := make([]net.Addr, len(addrs))
	for i, a := range addrs {
		ips[i] = &net.IPAddr{IP: net.ParseIP(a)}
	}
	i := &intf{addrs: ips}
	probeutils.InterfaceByName = func(name string) (probeutils.Addr, error) {
		if name != iname {
			return nil, errors.New("device not found")
		}
		return i, nil
	}
}

func TestInitSourceIP(t *testing.T) {
	rows := []struct {
		name       string
		sourceIP   string
		sourceIntf string
		intf       string
		intfAddrs  []string
		want       string
		wantError  bool
	}{
		{
			name:     "Use IP",
			sourceIP: "1.1.1.1",
			want:     "1.1.1.1",
		},
		{
			name:     "IP not set",
			sourceIP: "",
			want:     "",
		},
		{
			name:       "Interface with no adders fails",
			sourceIntf: "eth1",
			intf:       "eth1",
			wantError:  true,
		},
		{
			name:       "Unknown interface fails",
			sourceIntf: "eth1",
			intf:       "eth0",
			wantError:  true,
		},
		{
			name:       "Uses first addr for interface",
			sourceIntf: "eth1",
			intf:       "eth1",
			intfAddrs:  []string{"1.1.1.1", "2.2.2.2"},
			want:       "1.1.1.1",
		},
	}

	for _, r := range rows {
		c := &configpb.ProbeConf{}
		if r.sourceIP != "" {
			c.Source = &configpb.ProbeConf_SourceIp{r.sourceIP}
		} else if r.sourceIntf != "" {
			c.Source = &configpb.ProbeConf_SourceInterface{r.sourceIntf}
			mockInterfaceByName(r.intf, r.intfAddrs)
		}

		p := &Probe{}
		opts := &options.Options{
			Targets:   targets.StaticTargets("test.com"),
			Interval:  2 * time.Second,
			Timeout:   time.Second,
			ProbeConf: c,
		}
		err := p.Init("http_test", opts)

		if (err != nil) != r.wantError {
			t.Errorf("Row %q: newProbe() gave error %q, want error is %v", r.name, err, r.wantError)
			continue
		}
		if r.wantError {
			continue
		}
		if source, _ := p.getSourceFromConfig(); source != r.want {
			t.Errorf("Row %q: p.source = %q, want %q", r.name, source, r.want)
		}
	}
}
