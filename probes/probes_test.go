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
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes"
	"github.com/google/cloudprober/probes/options"
	configpb "github.com/google/cloudprober/probes/proto"
	"github.com/google/cloudprober/probes/testdata"
	"github.com/google/cloudprober/targets"
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
		Targets: &targets.TargetsDef{
			Type: &targets.TargetsDef_DummyTargets{},
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
	err := proto.SetExtension(probeDef, testdata.E_FancyProbe, &testdata.FancyProbe{Name: proto.String("fancy")})
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
	_, ok := probeMap["ext-probe"].(*testProbe)
	if !ok {
		t.Errorf("Extension probe (%v) is not of type *testProbe", probeMap["ext-probe"])
	}
	if testProbeIntialized != 1 {
		t.Errorf("Extensions probe's Init() called %d times, should be called exactly once.", testProbeIntialized)
	}
}
