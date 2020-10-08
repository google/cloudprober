// Copyright 2019 Google Inc.
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

package testutils

import (
	"testing"
	"time"

	"github.com/google/cloudprober/metrics"
)

func TestMetricsFromChannel(t *testing.T) {
	dataChan := make(chan *metrics.EventMetrics, 10)

	var ts [2]time.Time

	ts[0] = time.Now()
	time.Sleep(time.Millisecond)
	ts[1] = time.Now()

	// Put 2 EventMetrics, get 2
	dataChan <- metrics.NewEventMetrics(ts[0])
	dataChan <- metrics.NewEventMetrics(ts[1])

	ems, err := MetricsFromChannel(dataChan, 2, time.Second)
	if err != nil {
		t.Error(err)
	}

	for i := range ems {
		if ems[i].Timestamp != ts[i] {
			t.Errorf("First EventMetrics has unexpected timestamp. Got=%s, Expected=%s", ems[i], ts[i])
		}
	}

	// Put 2 EventMetrics, try to get 3
	dataChan <- metrics.NewEventMetrics(ts[0])
	dataChan <- metrics.NewEventMetrics(ts[1])

	ems, err = MetricsFromChannel(dataChan, 3, time.Second)
	if err == nil {
		t.Error("expected error got none")
	}

	for i := range ems {
		if ems[i].Timestamp != ts[i] {
			t.Errorf("First EventMetrics has unexpected timestamp. Got=%s, Expected=%s", ems[i], ts[i])
		}
	}
}

func TestMetricsMap(t *testing.T) {
	var ems []*metrics.EventMetrics
	expectedValues := map[string][]int64{
		"success": []int64{99, 98},
		"total":   []int64{100, 100},
	}
	ems = append(ems, metrics.NewEventMetrics(time.Now()).
		AddMetric("success", metrics.NewInt(99)).
		AddMetric("total", metrics.NewInt(100)).
		AddLabel("dst", "target1"))
	ems = append(ems, metrics.NewEventMetrics(time.Now()).
		AddMetric("success", metrics.NewInt(98)).
		AddMetric("total", metrics.NewInt(100)).
		AddLabel("dst", "target2"))

	metricsMap := MetricsMap(ems)

	for _, m := range []string{"success", "total"} {
		if metricsMap[m] == nil {
			t.Errorf("didn't get metric %s in metrics map", m)
		}
	}

	for i, tgt := range []string{"target1", "target2"} {
		for _, m := range []string{"success", "total"} {
			if len(metricsMap[m][tgt]) != 1 {
				t.Errorf("Wrong number of values for metric (%s) for target (%s) from the command output. Got=%d, Expected=1", m, tgt, len(metricsMap[m][tgt]))
			}
			val := metricsMap[m][tgt][0].Metric(m).(metrics.NumValue).Int64()
			if val != expectedValues[m][i] {
				t.Errorf("Wrong metric value for target (%s) from the command output. Got=%d, Expected=%d", m, val, expectedValues[m][i])
			}
		}
	}
}
