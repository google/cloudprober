// Copyright 2017-2019 The Cloudprober Authors.
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

package payload

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/metrics/payload/proto"
)

var (
	testPtype  = "external"
	testProbe  = "testprobe"
	testTarget = "test-target"
)

func parserForTest(t *testing.T, agg bool) *Parser {
	testConf := `
	  aggregate_in_cloudprober: %v
		dist_metric {
			key: "op_latency"
			value {
				explicit_buckets: "1,10,100"
			}
		}
 `
	var c configpb.OutputMetricsOptions
	if err := proto.UnmarshalText(fmt.Sprintf(testConf, agg), &c); err != nil {
		t.Error(err)
	}
	p, err := NewParser(&c, testPtype, testProbe, metrics.CUMULATIVE, nil)
	if err != nil {
		t.Error(err)
	}

	return p
}

// testData encapsulates the test data.
type testData struct {
	varA, varB float64
	lat        []float64
	labels     [3][2]string
}

// aggregatedEM returns an EventMetrics struct corresponding to the provided testData.
func (td *testData) aggregatedEM(ts time.Time) *metrics.EventMetrics {
	d := metrics.NewDistribution([]float64{1, 10, 100})
	for _, sample := range td.lat {
		d.AddSample(sample)
	}
	return metrics.NewEventMetrics(ts).
		AddMetric("op_latency", d).
		AddMetric("time_to_running", metrics.NewFloat(td.varA)).
		AddMetric("time_to_ssh", metrics.NewFloat(td.varB)).
		AddLabel("ptype", testPtype).
		AddLabel("probe", testProbe).
		AddLabel("dst", testTarget)
}

// aggregatedEM returns an EventMetrics struct corresponding to the provided testData.
func (td *testData) multiEM(ts time.Time) []*metrics.EventMetrics {
	var results []*metrics.EventMetrics

	d := metrics.NewDistribution([]float64{1, 10, 100})
	for _, sample := range td.lat {
		d.AddSample(sample)
	}

	results = append(results, []*metrics.EventMetrics{
		metrics.NewEventMetrics(ts).AddMetric("op_latency", d),
		metrics.NewEventMetrics(ts).AddMetric("time_to_running", metrics.NewFloat(td.varA)),
		metrics.NewEventMetrics(ts).AddMetric("time_to_ssh", metrics.NewFloat(td.varB)),
	}...)

	for i, em := range results {
		em.AddLabel("ptype", testPtype).
			AddLabel("probe", testProbe).
			AddLabel("dst", testTarget).
			AddLabel(td.labels[i][0], td.labels[i][1])
	}

	return results
}

func (td *testData) testPayload(quoteLabelValues bool) string {
	var labelStrs [3]string
	for i, kv := range td.labels {
		if kv[0] == "" {
			continue
		}
		if quoteLabelValues {
			labelStrs[i] = fmt.Sprintf("{%s=\"%s\"}", kv[0], kv[1])
		} else {
			labelStrs[i] = fmt.Sprintf("{%s=%s}", kv[0], kv[1])
		}
	}

	var latencyStrs []string
	for _, f := range td.lat {
		latencyStrs = append(latencyStrs, fmt.Sprintf("%f", f))
	}
	payloadLines := []string{
		fmt.Sprintf("op_latency%s %s", labelStrs[0], strings.Join(latencyStrs, ",")),
		fmt.Sprintf("time_to_running%s %f", labelStrs[1], td.varA),
		fmt.Sprintf("time_to_ssh%s %f", labelStrs[2], td.varB),
	}
	return strings.Join(payloadLines, "\n")
}

func testAggregatedPayloadMetrics(t *testing.T, em *metrics.EventMetrics, td, etd *testData) {
	t.Helper()

	expectedEM := etd.aggregatedEM(em.Timestamp)
	if em.String() != expectedEM.String() {
		t.Errorf("Output metrics not aggregated correctly:\nGot:      %s\nExpected: %s", em.String(), expectedEM.String())
	}
}

func testPayloadMetrics(t *testing.T, p *Parser, etd *testData) {
	t.Helper()

	ems := p.PayloadMetrics(etd.testPayload(false), testTarget)
	expectedMetrics := etd.multiEM(ems[0].Timestamp)
	for i, em := range ems {
		if em.String() != expectedMetrics[i].String() {
			t.Errorf("Output metrics not aggregated correctly:\nGot:      %s\nExpected: %s", em.String(), expectedMetrics[i].String())
		}
	}

	// Test with quoted label values
	ems = p.PayloadMetrics(etd.testPayload(true), testTarget)
	expectedMetrics = etd.multiEM(ems[0].Timestamp)
	for i, em := range ems {
		if em.String() != expectedMetrics[i].String() {
			t.Errorf("Output metrics not aggregated correctly:\nGot:      %s\nExpected: %s", em.String(), expectedMetrics[i].String())
		}
	}
}

func TestAggreagateInCloudprober(t *testing.T) {
	p := parserForTest(t, true)

	// First payload
	td := &testData{10, 30, []float64{3.1, 4.0, 13}, [3][2]string{}}
	em := p.AggregatedPayloadMetrics(nil, td.testPayload(false), testTarget)

	testAggregatedPayloadMetrics(t, em, td, td)

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

	em = p.AggregatedPayloadMetrics(em, td.testPayload(false), testTarget)
	testAggregatedPayloadMetrics(t, em, td, etd)
}

func TestNoAggregation(t *testing.T) {
	p := parserForTest(t, false)

	// First payload
	td := &testData{
		varA: 10,
		varB: 30,
		lat:  []float64{3.1, 4.0, 13},
		labels: [3][2]string{
			[2]string{"l1", "v1"},
			[2]string{"l2", "v2"},
			[2]string{"l3", "v3"},
		},
	}
	testPayloadMetrics(t, p, td)

	// Send another payload, cloudprober should not aggregate the metrics.
	td = &testData{
		varA: 8,
		varB: 45,
		lat:  []float64{6, 14.1, 2.1},
		labels: [3][2]string{
			[2]string{"l1", "v1"},
			[2]string{"l2", "v2"},
			[2]string{"l3", "v3"},
		},
	}
	testPayloadMetrics(t, p, td)
}

func TestMetricValueLabels(t *testing.T) {
	tests := []struct {
		desc    string
		line    string
		metric  string
		labels  [][2]string
		value   string
		wantErr bool
	}{
		{
			desc:   "metric with no labels",
			line:   "total 56",
			metric: "total",
			value:  "56",
		},
		{
			desc:   "metric with no labels, but more spaces",
			line:   "total   56",
			metric: "total",
			value:  "56",
		},
		{
			desc:   "standard metric with labels",
			line:   "total{service=serviceA,dc=xx} 56",
			metric: "total",
			labels: [][2]string{
				[2]string{"service", "serviceA"},
				[2]string{"dc", "xx"},
			},
			value: "56",
		},
		{
			desc:   "quoted labels, same result",
			line:   "total{service=\"serviceA\",dc=\"xx\"} 56",
			metric: "total",
			labels: [][2]string{
				[2]string{"service", "serviceA"},
				[2]string{"dc", "xx"},
			},
			value: "56",
		},
		{
			desc:   "a label value has space and more spaces",
			line:   "total{service=\"service A\", dc= \"xx\"} 56",
			metric: "total",
			labels: [][2]string{
				[2]string{"service", "service A"},
				[2]string{"dc", "xx"},
			},
			value: "56",
		},
		{
			desc:   "label and value have a space",
			line:   "version{service=\"service A\",dc=xx} \"version 1.5\"",
			metric: "version",
			labels: [][2]string{
				[2]string{"service", "service A"},
				[2]string{"dc", "xx"},
			},
			value: "\"version 1.5\"",
		},
		{
			desc: "only one brace, invalid line",
			line: "total{service=\"service A\",dc=\"xx\" 56",
		},
	}

	p := &Parser{}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			m, v, l := p.metricValueLabels(test.line)
			if m != test.metric {
				t.Errorf("Metric name: got=%s, wanted=%s", m, test.metric)
			}
			if v != test.value {
				t.Errorf("Metric value: got=%s, wanted=%s", v, test.value)
			}
			if !reflect.DeepEqual(l, test.labels) {
				t.Errorf("Metric labels: got=%v, wanted=%v", l, test.labels)
			}
		})
	}
}

func BenchmarkMetricValueLabels(b *testing.B) {
	payload := []string{
		"total 50",
		"total   56",
		"total{service=serviceA,dc=xx} 56",
		"total{service=\"serviceA\",dc=\"xx\"} 56",
		"version{service=\"service A\",dc=xx} \"version 1.5\"",
	}
	// run the em.String() function b.N times
	for n := 0; n < b.N; n++ {
		p := &Parser{}
		for _, s := range payload {
			p.metricValueLabels(s)
		}
	}
}
