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

package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

// probeRunResult captures the results of a single probe run. The way we work with
// stats makes sure that probeRunResult and its fields are not accessed concurrently
// (see documentation with statsKeeper below). That's the reason we use metrics.Int
// types instead of metrics.AtomicInt.
type probeRunResult struct {
	target string
	sent   metrics.Int
	rcvd   metrics.Int
	rtt    metrics.Int // microseconds
}

func newProbeRunResult(target string) probeRunResult {
	prr := probeRunResult{
		target: target,
	}
	// Borgmon and friends expect results to be in milliseconds. We should
	// change expectations at the Borgmon end once the transition to the new
	// metrics model is complete.
	prr.rtt.Str = func(i int64) string {
		return fmt.Sprintf("%.3f", float64(i)/1000)
	}
	return prr
}

// Metrics converts probeRunResult into a map of the metrics that is suitable for
// working with metrics.EventMetrics.
func (prr probeRunResult) Metrics() *metrics.EventMetrics {
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("sent", &prr.sent).
		AddMetric("rcvd", &prr.rcvd).
		AddMetric("rtt", &prr.rtt)
}

// Target returns the p.target.
func (prr probeRunResult) Target() string {
	return prr.target
}

func TestStatsKeeper(t *testing.T) {
	targets := []string{
		"target1",
		"target2",
	}
	pType := "test"
	pName := "testProbe"
	exportInterval := 2 * time.Second

	resultsChan := make(chan ProbeResult, len(targets))
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	targetsFunc := func() []string {
		return targets
	}
	dataChan := make(chan *metrics.EventMetrics, len(targets))

	go StatsKeeper(ctx, pType, pName, exportInterval, targetsFunc, resultsChan, dataChan, &logger.Logger{})

	for _, t := range targets {
		prr := newProbeRunResult(t)
		prr.sent.Inc()
		prr.rcvd.Inc()
		prr.rtt.IncBy(metrics.NewInt(20000))
		resultsChan <- prr
	}
	time.Sleep(3 * time.Second)
	close(dataChan)

	for em := range dataChan {
		var foundTarget bool
		for _, target := range targets {
			if em.Label("dst") == target {
				foundTarget = true
				break
			}
		}
		if !foundTarget {
			t.Error("didn't get expected target label in the event metric")
		}
		expectedValues := map[string]int64{
			"sent": 1,
			"rcvd": 1,
			"rtt":  20000,
		}
		for key, eVal := range expectedValues {
			val := em.Metric(key).(metrics.NumValue).Int64()
			if val != eVal {
				t.Errorf("%s metric is not set correctly. Got: %d, Expected: %d", key, val, eVal)
			}
		}
	}
}
