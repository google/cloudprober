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
	"time"

	"github.com/golang/protobuf/proto"
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

func TestStatsExportInterval(t *testing.T) {
	rows := []struct {
		name       string
		pType      *configpb.ProbeDef_Type
		interval   int32
		timeout    int32
		configured int32
		want       int
		wantError  bool
	}{
		{
			name:     "Interval bigger than default",
			interval: 15,
			timeout:  10,
			want:     15,
		},
		{
			name:     "Timeout bigger than interval",
			interval: 10,
			timeout:  12,
			want:     12,
		},
		{
			name:     "Interval and timeout less than default",
			interval: 2,
			timeout:  1,
			want:     int(defaultStatsExtportIntv.Seconds()),
		},
		{
			name:     "UDP probe: default twice of timeout- I",
			interval: 10,
			timeout:  12,
			pType:    configpb.ProbeDef_UDP.Enum(),
			want:     24,
		},
		{
			name:     "UDP probe: default twice of timeout - II",
			interval: 5,
			timeout:  6,
			pType:    configpb.ProbeDef_UDP.Enum(),
			want:     12,
		},
		{
			name:       "Error: stats export interval smaller than interval",
			interval:   2,
			timeout:    1,
			configured: 1,
			wantError:  true,
		},
		{
			name:       "Configured value is good",
			interval:   2,
			timeout:    1,
			configured: 10,
			want:       10,
		},
	}

	for _, r := range rows {
		p := &configpb.ProbeDef{
			Targets:      testTargets,
			IntervalMsec: proto.Int32(r.interval * 1000),
			TimeoutMsec:  proto.Int32(r.timeout * 1000),
		}

		if r.pType != nil {
			p.Type = r.pType
		}

		if r.configured != 0 {
			p.StatsExportIntervalMsec = proto.Int32(r.configured * 1000)
		}

		opts, err := BuildProbeOptions(p, nil, nil, nil)
		if (err != nil) != r.wantError {
			t.Errorf("Row %q: BuildProbeOptions() gave error %q, want error is %v", r.name, err, r.wantError)
			continue
		}
		if r.wantError {
			continue
		}

		want := time.Duration(r.want) * time.Second
		if opts.StatsExportInterval != want {
			t.Errorf("Unexpected stats export interval (test case: %s): want=%s, got=%s", r.name, want, opts.StatsExportInterval)
		}
	}
}
