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

package external

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/external/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets"
)

func testProbe(t *testing.T, agg bool) *Probe {
	p := &Probe{
		name:    "testprobe",
		c:       &configpb.ProbeConf{},
		opts:    &options.Options{Targets: targets.StaticTargets("testTarget1")},
		results: make(map[string]*result),
	}
	testConf := `
	  command: "/bin/true"
	  output_metrics_options {
            aggregate_in_cloudprober: %s
            dist_metric {
	      key: "op_latency"
	      value {
	        explicit_buckets: "1,10,100"
	      }
	   }
	}`
	aggStr := "false"
	if agg {
		aggStr = "true"
	}
	if err := proto.UnmarshalText(fmt.Sprintf(testConf, aggStr), p.c); err != nil {
		t.Error(err)
	}
	if err := p.initPayloadMetrics(); err != nil {
		t.Error(err)
	}

	p.updateTargets()
	return p
}

// testData encapsulates the test data.
type testData struct {
	varA, varB float64
	lat        []float64
}

// testEM returns an EventMetrics struct corresponding to the provided testData.
func testEM(ts time.Time, td *testData, target string) *metrics.EventMetrics {
	d := metrics.NewDistribution([]float64{1, 10, 100})
	for _, sample := range td.lat {
		d.AddSample(sample)
	}
	return metrics.NewEventMetrics(ts).
		AddMetric("op_latency", d).
		AddMetric("time_to_running", metrics.NewFloat(td.varA)).
		AddMetric("time_to_ssh", metrics.NewFloat(td.varB)).
		AddLabel("ptype", "external").
		AddLabel("probe", "testprobe").
		AddLabel("dst", target)
}

func testPayload(td *testData) string {
	var latencyStrs []string
	for _, f := range td.lat {
		latencyStrs = append(latencyStrs, fmt.Sprintf("%f", f))
	}
	payloadLines := []string{
		fmt.Sprintf("time_to_running %f", td.varA),
		fmt.Sprintf("time_to_ssh %f", td.varB),
		fmt.Sprintf("op_latency %s", strings.Join(latencyStrs, ",")),
	}
	return strings.Join(payloadLines, "\n")
}

func testPayloadMetrics(t *testing.T, p *Probe, td, etd *testData) {
	for _, target := range p.targets {
		tgt := target.Name
		em := p.payloadToMetrics(p.targets[0].Name, testPayload(td), p.results[tgt])
		expectedEM := testEM(em.Timestamp, etd, tgt)
		if em.String() != expectedEM.String() {
			t.Errorf("Output metrics not aggregated correctly:\nGot:      %s\nExpected: %s", em.String(), expectedEM.String())
		}
	}
}

func TestAggreagateInCloudprober(t *testing.T) {
	p := testProbe(t, true)

	// First payload
	td := &testData{10, 30, []float64{3.1, 4.0, 13}}
	testPayloadMetrics(t, p, td, td)

	// Send another payload, cloudprober should aggregate the metrics.
	oldtd := td
	td = &testData{
		varA: 8,
		varB: 45,
		lat:  []float64{6, 14.1, 2.1},
	}
	etd := &testData{
		varA: oldtd.varA + td.varA,
		varB: oldtd.varB + td.varB,
		lat:  append(oldtd.lat, td.lat...),
	}
	testPayloadMetrics(t, p, td, etd)
}

func TestNoAggregation(t *testing.T) {
	p := testProbe(t, false)

	// First payload
	td := &testData{10, 30, []float64{3.1, 4.0, 13}}
	testPayloadMetrics(t, p, td, td)

	// Send another payload, cloudprober should aggregate the metrics.
	td = &testData{
		varA: 8,
		varB: 45,
		lat:  []float64{6, 14.1, 2.1},
	}
	testPayloadMetrics(t, p, td, td)
}

func TestParseValue(t *testing.T) {
	var val string

	// Bad value, should return an error
	val = "234g"
	_, err := parseValue(val, false)
	if err == nil {
		t.Errorf("parseValue(%s) returned no error", val)
	}

	// Float value
	val = "234"
	v, err := parseValue(val, false)
	if err != nil {
		t.Errorf("parseValue(%s) returned error: %v", val, err)
	}
	if _, ok := v.(*metrics.Float); !ok {
		t.Errorf("parseValue(%s) returned a non-float: %v", val, v)
	}

	// String value, aggregation disabled
	val = "\"234\""
	v, err = parseValue(val, false)
	if err != nil {
		t.Errorf("parseValue(%s) returned error: %v", val, err)
	}
	if _, ok := v.(metrics.String); !ok {
		t.Errorf("parseValue(%s) returned a non-string: %v", val, v)
	}

	// String value, aggregation enabled = should fail
	v, err = parseValue(val, true)
	if err == nil {
		t.Errorf("parseValue(%s) returned no error for parsing a string value while aggregation is enabled", val)
	}

	// Map value
	val = "map:code 200:10 404:1"
	v, err = parseValue(val, false)
	if err != nil {
		t.Errorf("parseValue(%s) returned error: %v", val, err)
	}
	if _, ok := v.(*metrics.Map); !ok {
		t.Errorf("parseValue(%s) returned a non-map: %v", val, v)
	}

	// Dist value
	val = "dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12"
	v, err = parseValue(val, false)
	if err != nil {
		t.Errorf("parseValue(%s) returned error: %v", val, err)
	}
	if _, ok := v.(*metrics.Distribution); !ok {
		t.Errorf("parseValue(%s) returned a non-dist: %v", val, v)
	}
}
