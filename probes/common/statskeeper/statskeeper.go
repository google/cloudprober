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

/*
Package statskeeper implements utilities that are shared across multiple probe
types.
*/
package statskeeper

import (
	"context"
	"time"

	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets/endpoint"
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
// Typical StatsKeeper usage pattern is that the probes start a StatsKeeper
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
func StatsKeeper(ctx context.Context, ptype, name string, opts *options.Options, targetsFunc func() []endpoint.Endpoint, resultsChan <-chan ProbeResult, dataChan chan<- *metrics.EventMetrics) {
	targetMetrics := make(map[string]*metrics.EventMetrics)
	exportTicker := time.NewTicker(opts.StatsExportInterval)
	defer exportTicker.Stop()

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
				opts.Logger.Errorf("Error adding metrics from the probe result for the target: %s. Err: %v", t, err)
			}
		case ts := <-exportTicker.C:
			for _, t := range targetsFunc() {
				em := targetMetrics[t.Name]
				if em != nil {
					em.AddLabel("ptype", ptype)
					em.AddLabel("probe", name)
					em.AddLabel("dst", t.Name)
					em.Timestamp = ts

					em.LatencyUnit = opts.LatencyUnit

					for _, al := range opts.AdditionalLabels {
						em.AddLabel(al.KeyValueForTarget(t.Name))
					}

					if opts.LogMetrics != nil {
						opts.LogMetrics(em)
					}
					dataChan <- em.Clone()
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
