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
Package testutils provides utilities for tests.
*/
package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/google/cloudprober/metrics"
)

// MetricsFromChannel reads metrics.EventMetrics from dataChannel with a timeout
func MetricsFromChannel(dataChan chan *metrics.EventMetrics, num int, timeout time.Duration) (results []*metrics.EventMetrics, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var timedout bool

	for i := 0; i < num && !timedout; i++ {
		select {
		case em := <-dataChan:
			results = append(results, em)
		case <-ctx.Done():
			timedout = true
		}
	}

	if timedout {
		err = fmt.Errorf("timed out while waiting for data from dataChannel, got only %d metrics out of %d", len(results), num)
	}
	return
}

// MetricsMap rearranges a list of metrics into a map of map.
// {
//   "m1": {
//       "target1": [val1, val2..],
//       "target2": [val1],
//   },
//   "m2": {
//       ...
//   }
// }
func MetricsMap(ems []*metrics.EventMetrics) map[string]map[string][]*metrics.EventMetrics {
	results := make(map[string]map[string][]*metrics.EventMetrics)
	for _, em := range ems {
		target := em.Label("dst")
		for _, m := range em.MetricsKeys() {
			if results[m] == nil {
				results[m] = make(map[string][]*metrics.EventMetrics)
			}
			results[m][target] = append(results[m][target], em)
		}
	}
	return results
}
