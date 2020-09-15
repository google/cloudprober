// Copyright 2019-2020 Google Inc.
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

func TestHostWithPort(t *testing.T) {
	for _, test := range []struct {
		host         string
		port         int
		wantHostPort string
	}{
		{
			host:         "target1.ns.cluster.local",
			wantHostPort: "target1.ns.cluster.local",
		},
		{
			host:         "target1.ns.cluster.local",
			port:         8080,
			wantHostPort: "target1.ns.cluster.local:8080",
		},
	} {
		t.Run(fmt.Sprintf("host:%s,port:%d", test.host, test.port), func(t *testing.T) {
			hostPort := hostWithPort(test.host, test.port)
			if hostPort != test.wantHostPort {
				t.Errorf("hostPort: %s, want: %s", hostPort, test.wantHostPort)
			}
		})
	}
}

func TestURLHostAndHeaderForTarget(t *testing.T) {
	for _, test := range []struct {
		name            string
		fqdn            string
		probeHostHeader string
		port            int
		wantHostHeader  string
		wantURLHost     string
	}{
		{
			name:            "target1",
			fqdn:            "target1.ns.cluster.local",
			probeHostHeader: "svc.target",
			port:            8080,
			wantHostHeader:  "svc.target",
			wantURLHost:     "target1.ns.cluster.local",
		},
		{
			name:            "target1",
			fqdn:            "target1.ns.cluster.local",
			probeHostHeader: "",
			port:            8080,
			wantHostHeader:  "target1.ns.cluster.local:8080",
			wantURLHost:     "target1.ns.cluster.local",
		},
		{
			name:            "target1",
			fqdn:            "target1.ns.cluster.local",
			probeHostHeader: "",
			port:            0,
			wantHostHeader:  "target1.ns.cluster.local",
			wantURLHost:     "target1.ns.cluster.local",
		},
		{
			name:            "target1",
			fqdn:            "",
			probeHostHeader: "",
			port:            8080,
			wantHostHeader:  "target1:8080",
			wantURLHost:     "target1",
		},
		{
			name:            "target1",
			fqdn:            "",
			probeHostHeader: "",
			port:            0,
			wantHostHeader:  "target1",
			wantURLHost:     "target1",
		},
	} {
		t.Run(fmt.Sprintf("test:%+v", test), func(t *testing.T) {
			target := endpoint.Endpoint{
				Name:   test.name,
				Labels: map[string]string{"fqdn": test.fqdn},
			}

			hostHeader := hostHeaderForTarget(target, test.probeHostHeader, test.port)
			if hostHeader != test.wantHostHeader {
				t.Errorf("Got host header: %s, want header: %s", hostHeader, test.wantHostHeader)
			}

			urlHost := urlHostForTarget(target)
			if urlHost != test.wantURLHost {
				t.Errorf("Got URL host: %s, want URL host: %s", urlHost, test.wantURLHost)
			}
		})
	}
}

func TestRelURLforTarget(t *testing.T) {
	for _, test := range []struct {
		targetURLLabel string
		probeURL       string
		wantRelURL     string
	}{
		{
			// Both set, probe URL wins.
			targetURLLabel: "/target-url",
			probeURL:       "/metrics",
			wantRelURL:     "/metrics",
		},
		{
			// Only target label set.
			targetURLLabel: "/target-url",
			probeURL:       "",
			wantRelURL:     "/target-url",
		},
		{
			// Nothing set, we get nothing.
			targetURLLabel: "",
			probeURL:       "",
			wantRelURL:     "",
		},
	} {
		t.Run(fmt.Sprintf("test:%+v", test), func(t *testing.T) {
			target := endpoint.Endpoint{
				Name:   "test-target",
				Labels: map[string]string{relURLLabel: test.targetURLLabel},
			}

			relURL := relURLForTarget(target, test.probeURL)
			if relURL != test.wantRelURL {
				t.Errorf("Got URL: %s, want: %s", relURL, test.wantRelURL)
			}
		})
	}
}

// Following tests are more comprehensive tests for request URL and host header.
type testData struct {
	desc         string
	targetName   string
	targetFQDN   string
	resolveFirst bool
	probeHost    string
	wantURLHost  string
}

func createRequestAndVerify(t *testing.T, td testData, probePort, targetPort, expectedPort int, resolveF resolveFunc) {
	t.Helper()

	p := &Probe{
		protocol: "http",
		c: &configpb.ProbeConf{
			ResolveFirst: proto.Bool(td.resolveFirst),
		},
		opts: &options.Options{Targets: targets.StaticTargets(td.targetName)},
	}

	if probePort != 0 {
		p.c.Port = proto.Int32(int32(probePort))
	}

	if td.probeHost != "" {
		p.c.Headers = append(p.c.Headers, &configpb.ProbeConf_Header{
			Name:  proto.String("Host"),
			Value: proto.String(td.probeHost),
		})
	}

	target := endpoint.Endpoint{
		Name: td.targetName,
		Port: targetPort,
		Labels: map[string]string{
			"fqdn": td.targetFQDN,
		},
	}
	req := p.httpRequestForTarget(target, resolveF)

	wantURL := fmt.Sprintf("http://%s", hostWithPort(td.wantURLHost, expectedPort))
	if req.URL.String() != wantURL {
		t.Errorf("HTTP req URL: %s, wanted: %s", req.URL.String(), wantURL)
	}

	// Note that we test hostHeaderForTarget independently.
	wantHostHeader := hostHeaderForTarget(target, td.probeHost, expectedPort)
	if req.Host != wantHostHeader {
		t.Errorf("HTTP req.Host: %s, wanted: %s", req.Host, wantHostHeader)
	}
}

func testRequestHostAndURLWithDifferentPorts(t *testing.T, td testData) {
	t.Helper()

	var resolveF resolveFunc
	if td.resolveFirst {
		resolveF = func(target string, ipVer int) (net.IP, error) {
			return net.ParseIP(td.wantURLHost), nil
		}
	}

	for _, ports := range []struct {
		probePort    int
		targetPort   int
		expectedPort int
	}{
		{
			probePort:    0,
			targetPort:   0,
			expectedPort: 0,
		},
		{
			probePort:    8080,
			targetPort:   9313,
			expectedPort: 8080, // probe port wins
		},
		{
			probePort:    0,
			targetPort:   9313,
			expectedPort: 9313, // target port wins
		},
	} {
		t.Run(fmt.Sprintf("%s_probe_port_%d_endpoint_port_%d", td.desc, ports.probePort, ports.targetPort), func(t *testing.T) {
			createRequestAndVerify(t, td, ports.probePort, ports.targetPort, ports.expectedPort, resolveF)
		})
	}
}

func TestRequestHostAndURL(t *testing.T) {
	tests := []testData{
		{
			desc:        "no_resolve_first,no_probe_host_header",
			targetName:  "test-target.com",
			wantURLHost: "test-target.com",
		},
		{
			desc:        "no_resolve_first,fqdn,no_probe_host_header",
			targetName:  "test-target.com",
			targetFQDN:  "test.svc.cluster.local",
			wantURLHost: "test.svc.cluster.local",
		},
		{
			desc:        "no_resolve_first,host_header",
			targetName:  "test-target.com",
			probeHost:   "test-host",
			wantURLHost: "test-target.com",
		},
		{
			desc:         "resolve_first,no_probe_host_header",
			targetName:   "localhost",
			resolveFirst: true,
			wantURLHost:  "127.0.0.1",
		},
		{
			desc:         "resolve_first,probe_host_header",
			targetName:   "localhost",
			resolveFirst: true,
			probeHost:    "test-host",
			wantURLHost:  "127.0.0.1",
		},
	}

	for _, td := range tests {
		t.Run(td.desc, func(t *testing.T) {
			testRequestHostAndURLWithDifferentPorts(t, td)
		})
	}
}
