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

package probeutils

import (
	"context"
	"reflect"
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
	return probeRunResult{
		target: target,
	}
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

	for i := 0; i < len(dataChan); i++ {
		em := <-dataChan
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

func TestPayloadVerification(t *testing.T) {
	testBytes := []byte("test bytes")

	// Verify that for the larger payload sizes we get replicas of the same
	// bytes.
	for _, size := range []int{256, 999, 2048, 4 * len(testBytes)} {
		bytesBuf := PatternPayload(testBytes, size)

		var expectedBuf []byte
		for i := 0; i < size/len(testBytes); i++ {
			expectedBuf = append(expectedBuf, testBytes...)
		}
		// Pad 0s in the end.
		expectedBuf = append(expectedBuf, make([]byte, size-len(expectedBuf))...)
		if !reflect.DeepEqual(bytesBuf, expectedBuf) {
			t.Errorf("Bytes array:\n%o\n\nExpected:\n%o", bytesBuf, expectedBuf)
		}

		// Verify payload.
		err := VerifyPayloadPattern(bytesBuf, testBytes)
		if err != nil {
			t.Errorf("Data verification error: %v", err)
		}
	}
}

func benchmarkVerifyPayloadPattern(size int, b *testing.B) {
	testBytes := []byte("test bytes")
	bytesBuf := PatternPayload(testBytes, size)

	for n := 0; n < b.N; n++ {
		VerifyPayloadPattern(bytesBuf, testBytes)
	}
}

func BenchmarkVerifyPayloadPattern56(b *testing.B)   { benchmarkVerifyPayloadPattern(56, b) }
func BenchmarkVerifyPayloadPattern256(b *testing.B)  { benchmarkVerifyPayloadPattern(256, b) }
func BenchmarkVerifyPayloadPattern1999(b *testing.B) { benchmarkVerifyPayloadPattern(1999, b) }
func BenchmarkVerifyPayloadPattern9999(b *testing.B) { benchmarkVerifyPayloadPattern(9999, b) }
