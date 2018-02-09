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

package probes

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/testdata"
	"google3/go/context/context"
)

type testProbe struct{}

func (p *testProbe) Init(name string, opts *options.Options) error { return nil }

func (p *testProbe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {}

func TestGetExtensionProbe(t *testing.T) {
	probeDef := &ProbeDef{}

	// This has the same effect as using the following in your config:
	// probe {
	//    name: "run_fancy_probe_for_x"
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
	probeEx, _, err := getExtensionProbe(probeDef)
	if err == nil {
		t.Errorf("Expected error in building probe from extensions, got nil. probe: %v", probeEx)
	}

	// Register our test probe type and try again.
	RegisterProbeType(200, func() Probe {
		return &testProbe{}
	})
	probeEx, value, err := getExtensionProbe(probeDef)
	if err != nil {
		t.Errorf("Got error in building probe from extensions: %v", err)
	}
	_, ok := value.(*testdata.FancyProbe)
	if !ok {
		t.Errorf("Extension probe conf (%v) is not of type *testdata.FancyProbe", value)
	}
	_, ok = probeEx.(*testProbe)
	if !ok {
		t.Errorf("Extension probe (%v) is not of type *testProbe", probeEx)
	}
}
