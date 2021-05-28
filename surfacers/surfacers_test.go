// Copyright 2017 The Cloudprober Authors.
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

package surfacers

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	fileconfigpb "github.com/google/cloudprober/surfacers/file/proto"
	surfacerpb "github.com/google/cloudprober/surfacers/proto"
)

func TestDefaultConfig(t *testing.T) {
	s, err := Init(context.Background(), []*surfacerpb.SurfacerDef{})
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != len(defaultSurfacers) {
		t.Errorf("Didn't get default surfacers for no config")
	}
}

func TestEmptyConfig(t *testing.T) {
	s, err := Init(context.Background(), []*surfacerpb.SurfacerDef{&surfacerpb.SurfacerDef{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 0 {
		t.Errorf("Got surfacers for zero config: %v", s)
	}
}

func TestInferType(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatalf("error creating tempfile for test")
	}

	defer os.Remove(tmpfile.Name()) // clean up

	s, err := Init(context.Background(), []*surfacerpb.SurfacerDef{
		{
			Surfacer: &surfacerpb.SurfacerDef_FileSurfacer{
				&fileconfigpb.SurfacerConf{
					FilePath: proto.String(tmpfile.Name()),
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(s) != 1 {
		t.Errorf("len(s)=%d, expected=1", len(s))
	}

	if s[0].Type != "FILE" {
		t.Errorf("Surfacer type: %s, expected: FILE", s[0].Type)
	}
}

type testSurfacer struct {
	received []*metrics.EventMetrics
}

func (ts *testSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	ts.received = append(ts.received, em)
}

var testEventMetrics = []*metrics.EventMetrics{
	metrics.NewEventMetrics(time.Now()).
		AddMetric("total", metrics.NewInt(20)).
		AddMetric("timeout", metrics.NewInt(2)).
		AddLabel("ptype", "http").
		AddLabel("probe", "google_homepage"),
	metrics.NewEventMetrics(time.Now()).
		AddMetric("memory", metrics.NewInt(20)).
		AddMetric("num_goroutines", metrics.NewInt(2)).
		AddLabel("probe", "sysvars"),
}

func TestUserDefinedAndFiltering(t *testing.T) {
	ts1, ts2 := &testSurfacer{}, &testSurfacer{}
	Register("s1", ts1)
	Register("s2", ts2)

	configs := []*surfacerpb.SurfacerDef{
		{
			Name: proto.String("s1"),
			Type: surfacerpb.Type_USER_DEFINED.Enum(),
		},
		{
			Name: proto.String("s2"),
			IgnoreMetricsWithLabel: []*surfacerpb.LabelFilter{
				{
					Key:   proto.String("probe"),
					Value: proto.String("sysvars"),
				},
			},
			Type: surfacerpb.Type_USER_DEFINED.Enum(),
		},
	}
	wantSurfacers := []string{"s1", "s2"}

	si, err := Init(context.Background(), configs)
	if err != nil {
		t.Fatalf("Unexpected initialization error: %v", err)
	}
	for i, s := range si {
		if s.Name != wantSurfacers[i] {
			t.Errorf("Got surfacer: %s, want surfacer: %s", s.Name, wantSurfacers[i])
		}
	}

	for _, em := range testEventMetrics {
		for _, s := range si {
			s.Surfacer.Write(context.Background(), em)
		}
	}

	wantEventMetrics := [][]*metrics.EventMetrics{
		testEventMetrics,      // No filtering.
		testEventMetrics[0:1], // One EM is ignored for the 2nd surfacer.
	}

	for i, ts := range []*testSurfacer{ts1, ts2} {
		wantEMs := wantEventMetrics[i]
		if !reflect.DeepEqual(ts.received, wantEMs) {
			t.Errorf("ts[%d]: Received EventMetrics: %v, want EventMetrics: %v", i, ts.received, wantEMs)
		}
	}
}
