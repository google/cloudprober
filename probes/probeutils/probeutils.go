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
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

// ProbeResult represents results of a probe run.
type ProbeResult interface {
	// Metrics converts ProbeResult into a map of the metrics that is suitable for
	// working with metrics.EventMetrics.
	Metrics() *metrics.EventMetrics

	// Target returns the target associated with the probe result.
	Target() string
}

// StatsKeeper manages and outputs probe results.
//
// Typical StatsKeepr usage pattern is that the probes start a StatsKeeper
// goroutine in the beginning. StatsKeeper goroutine manages access to the
// per-target cumulative metrics. It listens on an input channel for probe
// results and updates the metrics whenever a new probe result is obtained.
// It exports aggregate probe statistics to the output channel, at intervals
// controlled by a Ticker. These two operations are mutually exclusive. This
// is the only goroutine that accesses the metrics. StatsKeeper runs
// indefinitely, across multiple probe runs, and should not stop during normal
// program execution.
//
// If we get a new result on resultsChan, update the probe statistics.
// If we get a timer tick on doExport, export probe data for all targets.
// If context is canceled, return.
//
// Note that StatsKeeper calls a function (targetsFunc) to get the list of the
// targets for exporting results,  instead of getting a static list in the
// arguments. We do that as the list of targets is usually dynamic and is
// updated on a regular basis.
func StatsKeeper(ctx context.Context, ptype, name string, exportInterval time.Duration, targetsFunc func() []string, resultsChan <-chan ProbeResult, dataChan chan<- *metrics.EventMetrics, l *logger.Logger) {
	targetMetrics := make(map[string]*metrics.EventMetrics)
	doExport := time.Tick(exportInterval)

	for {
		select {
		case result := <-resultsChan:
			// result is a ProbeResult
			t := result.Target()
			if targetMetrics[t] == nil {
				targetMetrics[t] = result.Metrics()
				continue
			}
			err := targetMetrics[t].Update(result.Metrics())
			if err != nil {
				l.Errorf("Error adding metrics from the probe result for the target: %s. Err: %v", t, err)
			}
		case ts := <-doExport:
			for _, t := range targetsFunc() {
				em := targetMetrics[t]
				if em != nil {
					em.AddLabel("ptype", ptype)
					em.AddLabel("probe", name)
					em.AddLabel("dst", t)
					em.Timestamp = ts
					l.Info(em.String())
					dataChan <- em.Clone()
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// PatternPayload builds a payload that can be verified using VerifyPayloadPattern.
// It repeats the pattern to fill a []byte slice of finalSize. Last remaining
// bytes (finalSize mod patternSize) are left unpopulated (hence set to 0
// bytes). If final payload size is smaller than the pattern size, we return
// the pattern unmodified.
func PatternPayload(pattern []byte, finalSize int) []byte {
	if len(pattern) >= finalSize {
		return pattern
	}
	b := make([]byte, finalSize)
	patternSize := len(pattern)
	// We create finalSize/patternSize replicas of the payload.
	for nReplica := 0; nReplica < finalSize/patternSize; nReplica++ {
		copy(b[nReplica*patternSize:], pattern)
	}
	return b
}

// VerifyPayloadPattern verifies the payload built using PatternPayload.
func VerifyPayloadPattern(payload, pattern []byte) error {
	patternSize := len(pattern)
	nReplica := len(payload) / patternSize

	for i := 0; i < nReplica; i++ {
		bN := payload[0:patternSize]    // Next pattern sized bytes
		payload = payload[patternSize:] // Shift payload for next iteration

		if bytes.Compare(bN, pattern) != 0 {
			return fmt.Errorf("bytes are not in the expected format. payload[%d-Replica]=%v, pattern=%v", i, bN, pattern)
		}
	}

	// Verity that remaining bytes in payload are all zeros.
	if bytes.Compare(payload, make([]byte, len(payload))) != 0 {
		return fmt.Errorf("payload doesn't have 0s padding in the last 'payloadSize mod patternSize' bytes: %v", payload)
	}
	return nil
}
