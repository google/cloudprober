// Copyright 2019 Google Inc.
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
	"testing"

	"github.com/golang/protobuf/proto"
	configpb "github.com/google/cloudprober/probes/http/proto"
	"github.com/google/cloudprober/targets/endpoint"
)

func newProbe(port int) *Probe {
	p := &Probe{
		protocol: "http",
	}

	if port != 0 {
		p.c = &configpb.ProbeConf{
			Port: proto.Int(port),
		}
	}

	return p
}

func testReq(t *testing.T, p *Probe, ep endpoint.Endpoint, wantURL, wantHost string) {
	t.Helper()

	req := p.httpRequestForTarget(ep)
	if req.URL.String() != wantURL {
		t.Errorf("HTTP req URL: %s, wanted: %s", req.URL.String(), wantURL)
	}
}

func TestHTTPRequestForTarget(t *testing.T) {
	// Probe has no port
	p := newProbe(0)
	ep := endpoint.Endpoint{
		Name: "test-target.com",
	}
	testReq(t, p, ep, "http://test-target.com", ep.Name)

	// Probe and endpoint have port, probe's port wins.
	p = newProbe(8080)
	ep = endpoint.Endpoint{
		Name: "test-target.com",
		Port: 9313,
	}
	testReq(t, p, ep, "http://test-target.com:8080", ep.Name)

	// Only endpoint has port, endpoint's port is used.
	p = newProbe(0)
	ep = endpoint.Endpoint{
		Name: "test-target.com",
		Port: 9313,
	}
	testReq(t, p, ep, "http://test-target.com:9313", ep.Name)
}
