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

package options

import (
	"errors"
	"net"
	"testing"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/probes/probeutils"
	configpb "github.com/google/cloudprober/probes/proto"
	targetspb "github.com/google/cloudprober/targets/proto"
)

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

var ipVersionToEnum = map[int]*configpb.ProbeDef_IPVersion{
	4: configpb.ProbeDef_IPV4.Enum(),
	6: configpb.ProbeDef_IPV6.Enum(),
}

func TestGetSourceIPFromConfig(t *testing.T) {
	rows := []struct {
		name       string
		sourceIP   string
		sourceIntf string
		intf       string
		intfAddrs  []string
		ipVer      int
		want       string
		wantError  bool
	}{
		{
			name:     "Use IP",
			sourceIP: "1.1.1.1",
			want:     "1.1.1.1",
		},
		{
			name:      "Source IP doesn't match IP version",
			sourceIP:  "1.1.1.1",
			ipVer:     6,
			wantError: true,
		},
		{
			name:     "Use IPv6",
			sourceIP: "::1",
			ipVer:    6,
			want:     "::1",
		},
		{
			name:      "Invalid IP",
			sourceIP:  "12ab",
			wantError: true,
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
		{
			name:       "Uses first IPv6 addr for interface",
			sourceIntf: "eth1",
			intf:       "eth1",
			intfAddrs:  []string{"1.1.1.1", "::1"},
			ipVer:      6,
			want:       "::1",
		},
	}

	for _, r := range rows {
		p := &configpb.ProbeDef{
			IpVersion: ipVersionToEnum[r.ipVer],
		}

		if r.sourceIP != "" {
			p.SourceIpConfig = &configpb.ProbeDef_SourceIp{r.sourceIP}
		} else if r.sourceIntf != "" {
			p.SourceIpConfig = &configpb.ProbeDef_SourceInterface{r.sourceIntf}
			mockInterfaceByName(r.intf, r.intfAddrs)
		}

		source, err := getSourceIPFromConfig(p, &logger.Logger{})

		if (err != nil) != r.wantError {
			t.Errorf("Row %q: getSourceIPFromConfig() gave error %q, want error is %v", r.name, err, r.wantError)
			continue
		}
		if r.wantError {
			continue
		}
		if source.String() != r.want {
			t.Errorf("Row %q: source= %q, want %q", r.name, source, r.want)
		}
	}
}

var testTargets = &targetspb.TargetsDef{
	Type: &targetspb.TargetsDef_HostNames{HostNames: "testHost"},
}

func TestIPVersionFromSourceIP(t *testing.T) {
	rows := []struct {
		name     string
		sourceIP string
		ipVer    int
	}{
		{
			name:  "No source IP",
			ipVer: 0,
		},
		{
			name:     "IPv4 from source IP",
			sourceIP: "1.1.1.1",
			ipVer:    4,
		},
		{
			name:     "IPv6 from source IP",
			sourceIP: "::1",
			ipVer:    6,
		},
	}

	for _, r := range rows {
		p := &configpb.ProbeDef{
			Targets: testTargets,
		}

		if r.sourceIP != "" {
			p.SourceIpConfig = &configpb.ProbeDef_SourceIp{r.sourceIP}
		}

		opts, err := BuildProbeOptions(p, nil, nil, nil)
		if err != nil {
			t.Errorf("got unexpected error: %v", err)
			continue
		}

		if opts.IPVersion != r.ipVer {
			t.Errorf("Unexpected IPVersion (test case: %s) want=%d, got=%d", r.name, r.ipVer, opts.IPVersion)
		}
	}
}
