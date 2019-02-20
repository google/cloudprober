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

package probes_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/probeutils"
	configpb "github.com/google/cloudprober/probes/proto"
	testdatapb "github.com/google/cloudprober/probes/testdata"
	targetspb "github.com/google/cloudprober/targets/proto"
)

var testProbeIntialized int

type testProbe struct{}

func (p *testProbe) Init(name string, opts *options.Options) error {
	testProbeIntialized++
	return nil
}

func (p *testProbe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {}

func TestGetExtensionProbe(t *testing.T) {
	probeDef := &configpb.ProbeDef{
		Name: proto.String("ext-probe"),
		Type: configpb.ProbeDef_EXTENSION.Enum(),
		Targets: &targetspb.TargetsDef{
			Type: &targetspb.TargetsDef_DummyTargets{},
		},
	}

	// This has the same effect as using the following in your config:
	// probe {
	//    name: "ext-probe"
	//    targets: ...
	//    ...
	//    [cloudprober.probes.testdata.fancy_probe] {
	//      name: "fancy"
	//    }
	// }
	err := proto.SetExtension(probeDef, testdatapb.E_FancyProbe, &testdatapb.FancyProbe{Name: proto.String("fancy")})
	if err != nil {
		t.Fatalf("error setting up extension in test probe proto: %v", err)
	}
	probeMap, err := probes.Init([]*configpb.ProbeDef{probeDef}, nil, &logger.Logger{}, make(map[string]string))
	if err == nil {
		t.Errorf("Expected error in building probe from extensions, got nil. Probes map: %v", probeMap)
	}
	t.Log(err.Error())

	// Register our test probe type and try again.
	probes.RegisterProbeType(200, func() probes.Probe {
		return &testProbe{}
	})

	probeMap, err = probes.Init([]*configpb.ProbeDef{probeDef}, nil, &logger.Logger{}, make(map[string]string))
	if err != nil {
		t.Errorf("Got error in building probe from extensions: %v", err)
	}
	if probeMap["ext-probe"] == nil {
		t.Errorf("Extension probe not in the probes map")
	}
	_, ok := probeMap["ext-probe"].Probe.(*testProbe)
	if !ok {
		t.Errorf("Extension probe (%v) is not of type *testProbe", probeMap["ext-probe"])
	}
	if testProbeIntialized != 1 {
		t.Errorf("Extensions probe's Init() called %d times, should be called exactly once.", testProbeIntialized)
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

	tProbe := &testProbe{}
	probes.RegisterUserDefined("test-init-source-ip", tProbe)

	for _, r := range rows {
		probeDef := &configpb.ProbeDef{
			Name: proto.String("test-init-source-ip"),
			Type: configpb.ProbeDef_USER_DEFINED.Enum(),
			Targets: &targetspb.TargetsDef{
				Type: &targetspb.TargetsDef_DummyTargets{},
			},
		}

		if r.sourceIP != "" {
			probeDef.SourceIpConfig = &configpb.ProbeDef_SourceIp{SourceIp: r.sourceIP}
		} else if r.sourceIntf != "" {
			probeDef.SourceIpConfig = &configpb.ProbeDef_SourceInterface{SourceInterface: r.sourceIntf}
			mockInterfaceByName(r.intf, r.intfAddrs)
		}

		probeMap, err := probes.Init([]*configpb.ProbeDef{probeDef}, nil, &logger.Logger{}, make(map[string]string))

		if (err != nil) != r.wantError {
			t.Errorf("Row %q: newProbe() gave error %q, want error is %v", r.name, err, r.wantError)
			continue
		}
		if r.wantError {
			continue
		}

		p := probeMap["test-init-source-ip"]
		if p == nil {
			t.Errorf("Row %q: probes.Init() returned nil probe", r.name)
		}

		if p.SourceIP != r.want {
			t.Errorf("Row %q: p.source = %q, want %q", r.name, p.SourceIP, r.want)
		}
	}
}
