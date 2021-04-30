// Copyright 2017-2020 The Cloudprober Authors.
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

package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/surfacers/prometheus/proto"
)

func newEventMetrics(sent, rcvd int64, respCodes map[string]int64, ptype, probe string) *metrics.EventMetrics {
	respCodesVal := metrics.NewMap("code", metrics.NewInt(0))
	for k, v := range respCodes {
		respCodesVal.IncKeyBy(k, metrics.NewInt(v))
	}
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("sent", metrics.NewInt(sent)).
		AddMetric("rcvd", metrics.NewInt(rcvd)).
		AddMetric("resp-code", respCodesVal).
		AddLabel("ptype", ptype).
		AddLabel("probe", probe)
}

func verify(t *testing.T, ps *PromSurfacer, expectedMetrics map[string]testData) {
	for k, td := range expectedMetrics {
		pm := ps.metrics[td.metricName]
		if pm == nil {
			t.Errorf("Metric %s not found in the prometheus metrics: %v", k, ps.metrics)
			continue
		}
		if pm.data[k] == nil {
			t.Errorf("Data key %s not found in the prometheus metrics: %v", k, pm.data)
			continue
		}
		if pm.data[k].value != td.value {
			t.Errorf("Didn't get expected metrics. Got: %s, Expected: %s", pm.data[k].value, td.value)
		}
	}
	var dataCount int
	for _, pm := range ps.metrics {
		dataCount += len(pm.data)
	}
	if dataCount != len(expectedMetrics) {
		t.Errorf("Prometheus doesn't have expected number of data keys. Got: %d, Expected: %d", dataCount, len(expectedMetrics))
	}
}

// mergeMap is helper function to build expectedMetrics by merging newly
// added expectedMetrics with the existing ones.
func mergeMap(recv map[string]testData, newmap map[string]testData) {
	for k, v := range newmap {
		recv[k] = v
	}
}

// testData encapsulates expected value for a metric key and metric name.
type testData struct {
	metricName string // To access data row in a 2-level data structure.
	value      string
}

func newPromSurfacer(t *testing.T, writeTimestamp bool) *PromSurfacer {
	c := &configpb.SurfacerConf{
		// Attach a random integer to metrics URL so that multiple
		// tests can run in parallel without handlers clashing with
		// each other.
		MetricsUrl:       proto.String(fmt.Sprintf("/metrics_%d", rand.Int())),
		IncludeTimestamp: proto.Bool(writeTimestamp),
	}
	l, _ := logger.New(context.Background(), "promtheus_test")
	ps, err := New(context.Background(), c, l)
	if err != nil {
		t.Fatal("Error while initializing prometheus surfacer", err)
	}
	return ps
}

func TestRecord(t *testing.T) {
	ps := newPromSurfacer(t, true)

	// Record first EventMetrics
	ps.record(newEventMetrics(32, 22, map[string]int64{
		"200": 22,
	}, "http", "vm-to-google"))
	expectedMetrics := map[string]testData{
		"sent{ptype=\"http\",probe=\"vm-to-google\"}":                   testData{"sent", "32"},
		"rcvd{ptype=\"http\",probe=\"vm-to-google\"}":                   testData{"rcvd", "22"},
		"resp_code{ptype=\"http\",probe=\"vm-to-google\",code=\"200\"}": testData{"resp_code", "22"},
	}
	verify(t, ps, expectedMetrics)

	// Record second EventMetrics, no overlap.
	ps.record(newEventMetrics(500, 492, map[string]int64{}, "ping", "vm-to-vm"))
	mergeMap(expectedMetrics, map[string]testData{
		"sent{ptype=\"ping\",probe=\"vm-to-vm\"}": testData{"sent", "500"},
		"rcvd{ptype=\"ping\",probe=\"vm-to-vm\"}": testData{"rcvd", "492"},
	})
	verify(t, ps, expectedMetrics)

	// Record third EventMetrics, replaces first EventMetrics' metrics.
	ps.record(newEventMetrics(62, 50, map[string]int64{
		"200": 42,
		"204": 8,
	}, "http", "vm-to-google"))
	mergeMap(expectedMetrics, map[string]testData{
		"sent{ptype=\"http\",probe=\"vm-to-google\"}":                   testData{"sent", "62"},
		"rcvd{ptype=\"http\",probe=\"vm-to-google\"}":                   testData{"rcvd", "50"},
		"resp_code{ptype=\"http\",probe=\"vm-to-google\",code=\"200\"}": testData{"resp_code", "42"},
		"resp_code{ptype=\"http\",probe=\"vm-to-google\",code=\"204\"}": testData{"resp_code", "8"},
	})
	verify(t, ps, expectedMetrics)

	// Test string metrics.
	em := metrics.NewEventMetrics(time.Now()).
		AddMetric("instance_id", metrics.NewString("23152113123131")).
		AddMetric("version", metrics.NewString("cloudradar-20170606-RC00")).
		AddLabel("module", "sysvars")
	em.Kind = metrics.GAUGE
	ps.record(em)
	mergeMap(expectedMetrics, map[string]testData{
		"instance_id{module=\"sysvars\",val=\"23152113123131\"}":       testData{"instance_id", "1"},
		"version{module=\"sysvars\",val=\"cloudradar-20170606-RC00\"}": testData{"version", "1"},
	})
	verify(t, ps, expectedMetrics)
}

func TestInvalidNames(t *testing.T) {
	ps := newPromSurfacer(t, true)
	respCodesVal := metrics.NewMap("resp-code", metrics.NewInt(0))
	respCodesVal.IncKeyBy("200", metrics.NewInt(19))
	ps.record(metrics.NewEventMetrics(time.Now()).
		AddMetric("sent", metrics.NewInt(32)).
		AddMetric("rcvd/sent", metrics.NewInt(22)).
		AddMetric("resp", respCodesVal).
		AddLabel("probe-type", "http").
		AddLabel("probe/name", "vm-to-google"))

	// Metric rcvd/sent is dropped
	// Label probe-type is converted to probe_type
	// Label probe/name is dropped
	// Map value key resp-code is converted to resp_code label name
	expectedMetrics := map[string]testData{
		"sent{probe_type=\"http\"}":                   testData{"sent", "32"},
		"resp{probe_type=\"http\",resp_code=\"200\"}": testData{"resp", "19"},
	}
	verify(t, ps, expectedMetrics)
}

func TestScrapeOutput(t *testing.T) {
	ps := newPromSurfacer(t, true)
	respCodesVal := metrics.NewMap("code", metrics.NewInt(0))
	respCodesVal.IncKeyBy("200", metrics.NewInt(19))
	latencyVal := metrics.NewDistribution([]float64{1, 4})
	latencyVal.AddSample(0.5)
	latencyVal.AddSample(5)
	ts := time.Now()
	promTS := fmt.Sprintf("%d", ts.UnixNano()/(1000*1000))
	ps.record(metrics.NewEventMetrics(ts).
		AddMetric("sent", metrics.NewInt(32)).
		AddMetric("rcvd", metrics.NewInt(22)).
		AddMetric("latency", latencyVal).
		AddMetric("resp_code", respCodesVal).
		AddLabel("ptype", "http"))
	var b bytes.Buffer
	ps.writeData(&b)
	data := b.String()
	for _, d := range []string{
		"#TYPE sent counter",
		"#TYPE rcvd counter",
		"#TYPE resp_code counter",
		"#TYPE latency histogram",
		"sent{ptype=\"http\"} 32 " + promTS,
		"rcvd{ptype=\"http\"} 22 " + promTS,
		"resp_code{ptype=\"http\",code=\"200\"} 19 " + promTS,
		"latency_sum{ptype=\"http\"} 5.5 " + promTS,
		"latency_count{ptype=\"http\"} 2 " + promTS,
		"latency_bucket{ptype=\"http\",le=\"1\"} 1 " + promTS,
		"latency_bucket{ptype=\"http\",le=\"4\"} 1 " + promTS,
		"latency_bucket{ptype=\"http\",le=\"+Inf\"} 2 " + promTS,
	} {
		if strings.Index(data, d) == -1 {
			t.Errorf("String \"%s\" not found in output data: %s", d, data)
		}
	}
}

func TestScrapeOutputNoTimestamp(t *testing.T) {
	ps := newPromSurfacer(t, false)
	respCodesVal := metrics.NewMap("code", metrics.NewInt(0))
	respCodesVal.IncKeyBy("200", metrics.NewInt(19))
	latencyVal := metrics.NewDistribution([]float64{1, 4})
	latencyVal.AddSample(0.5)
	latencyVal.AddSample(5)
	ps.record(metrics.NewEventMetrics(time.Now()).
		AddMetric("sent", metrics.NewInt(32)).
		AddMetric("rcvd", metrics.NewInt(22)).
		AddMetric("latency", latencyVal).
		AddMetric("resp_code", respCodesVal).
		AddLabel("ptype", "http"))
	var b bytes.Buffer
	ps.writeData(&b)
	data := b.String()
	for _, d := range []string{
		"#TYPE sent counter",
		"#TYPE rcvd counter",
		"#TYPE resp_code counter",
		"#TYPE latency histogram",
		"sent{ptype=\"http\"} 32",
		"rcvd{ptype=\"http\"} 22",
		"resp_code{ptype=\"http\",code=\"200\"} 19",
		"latency_sum{ptype=\"http\"} 5.5",
		"latency_count{ptype=\"http\"} 2",
		"latency_bucket{ptype=\"http\",le=\"1\"} 1",
		"latency_bucket{ptype=\"http\",le=\"4\"} 1",
		"latency_bucket{ptype=\"http\",le=\"+Inf\"} 2",
	} {
		if strings.Index(data, d) == -1 {
			t.Errorf("String \"%s\" not found in output data: %s", d, data)
		}
	}
}
