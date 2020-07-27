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
	"fmt"
	"net"
	"testing"

	"github.com/golang/protobuf/proto"
	configpb "github.com/google/cloudprober/probes/http/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/targets/endpoint"
)

func newProbe(port int, resolveFirst bool, target, hostHeader string) *Probe {
	p := &Probe{
		protocol: "http",
		c: &configpb.ProbeConf{
			ResolveFirst: proto.Bool(resolveFirst),
		},
		opts: &options.Options{Targets: targets.StaticTargets(target)},
	}

	if port != 0 {
		p.c.Port = proto.Int(port)
	}

	if hostHeader != "" {
		p.c.Headers = append(p.c.Headers, &configpb.ProbeConf_Header{
			Name:  proto.String("Host"),
			Value: proto.String(hostHeader),
		})
	}

	return p
}

func hostWithPort(host string, port int) string {
	if port == 0 {
		return host
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func testReq(t *testing.T, p *Probe, ep endpoint.Endpoint, wantURLHost, hostHeader string, port int, resolveF resolveFunc) {
	t.Helper()

	wantURL := fmt.Sprintf("http://%s", hostWithPort(wantURLHost, port))
	wantHost := hostHeader
	if wantHost == "" {
		wantHost = hostWithPort(ep.Name, port)
	}

	req := p.httpRequestForTarget(ep, resolveF)
	if req.URL.String() != wantURL {
		t.Errorf("HTTP req URL: %s, wanted: %s", req.URL.String(), wantURL)
	}

	if req.Host != wantHost {
		t.Errorf("HTTP req.Host: %s, wanted: %s", req.Host, wantHost)
	}
}

func testRequestHostAndURL(t *testing.T, resolveFirst bool, target, urlHost, hostHeader string) {
	t.Helper()

	var resolveF resolveFunc
	if resolveFirst {
		resolveF = func(target string, ipVer int) (net.IP, error) {
			return net.ParseIP(urlHost), nil
		}
	}

	// Probe has no port
	p := newProbe(0, resolveFirst, target, hostHeader)
	expectedPort := 0

	// If hostHeader is set, change probe config and expectedHost.
	ep := endpoint.Endpoint{
		Name: target,
	}
	testReq(t, p, ep, urlHost, hostHeader, expectedPort, resolveF)

	// Probe and endpoint have port, probe's port wins.
	p = newProbe(8080, resolveFirst, target, hostHeader)
	ep = endpoint.Endpoint{
		Name: target,
		Port: 9313,
	}
	expectedPort = 8080
	testReq(t, p, ep, urlHost, hostHeader, expectedPort, resolveF)

	// Only endpoint has port, endpoint's port is used.
	p = newProbe(0, resolveFirst, target, hostHeader)
	ep = endpoint.Endpoint{
		Name: target,
		Port: 9313,
	}
	expectedPort = 9313
	testReq(t, p, ep, urlHost, hostHeader, expectedPort, resolveF)
}

func TestRequestHostAndURL(t *testing.T) {
	t.Run("no_resolve_first,no_host_header", func(t *testing.T) {
		testRequestHostAndURL(t, false, "test-target.com", "test-target.com", "")
	})

	t.Run("no_resolve_first,host_header", func(t *testing.T) {
		testRequestHostAndURL(t, false, "test-target.com", "test-target.com", "test-host")
	})

	t.Run("resolve_first,no_host_header", func(t *testing.T) {
		testRequestHostAndURL(t, true, "localhost", "127.0.0.1", "")
	})

	t.Run("resolve_first,host_header", func(t *testing.T) {
		testRequestHostAndURL(t, true, "localhost", "127.0.0.1", "test-host")
	})
}
