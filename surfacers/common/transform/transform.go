// Copyright 2021 The Cloudprober Authors.
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

// Package transform implements some transformations for metrics before we
// export them.
package transform

import (
	"fmt"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

// AddFailureMetric adds failure metric to the EventMetrics based on the
// config options.
func AddFailureMetric(em *metrics.EventMetrics) error {
	tv, sv, fv := em.Metric("total"), em.Metric("success"), em.Metric("failure")
	// If there is already a failure metric, or if "total" and "success" metrics
	// are not available, don't compute failure metric.
	if fv != nil || tv == nil || sv == nil {
		return nil
	}

	total, totalOk := tv.(metrics.NumValue)
	success, successOk := sv.(metrics.NumValue)
	if !totalOk || !successOk {
		return fmt.Errorf("total (%v) and success (%v) values are not numeric, this should never happen", tv, sv)
	}

	em.AddMetric("failure", metrics.NewInt(total.Int64()-success.Int64()))
	return nil
}

// CumulativeToGauge creates a "gauge" EventMetrics from a "cumulative"
// EventMetrics using a cache. It looks for the EventMetrics in the given cache
// and if it exists already, it subtracts the current values from the cached
// values.
func CumulativeToGauge(em *metrics.EventMetrics, lvCache map[string]*metrics.EventMetrics, l *logger.Logger) (*metrics.EventMetrics, error) {
	key := em.Key()

	lastEM, ok := lvCache[key]
	// Cache a copy of "em" as some fields like maps and dist can be shared
	// across successive "em" writes.
	lvCache[key] = em.Clone()

	// If it is the first time for this EventMetrics, return it as it is.
	if !ok {
		return em, nil
	}

	gaugeEM, err := em.SubtractLast(lastEM)
	if err != nil {
		return nil, fmt.Errorf("error subtracting cached metrics from current metrics: %v", err)
	}

	return gaugeEM, nil
}
