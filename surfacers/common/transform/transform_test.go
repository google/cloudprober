// Copyright 2017-2021 The Cloudprober Authors.
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

package transform

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/cloudprober/metrics"
)

func TestXxx(t *testing.T) {
}

func TestFailureCountForDefaultMetrics(t *testing.T) {
	var tests = []struct {
		total, success, failure metrics.Value
		wantFailure             *metrics.Int
		wantErr                 bool
	}{
		{
			total:       metrics.NewInt(1000),
			success:     metrics.NewInt(990),
			wantFailure: metrics.NewInt(10),
		},
		{
			total:       metrics.NewInt(1000),
			success:     metrics.NewInt(990),
			failure:     metrics.NewInt(20),
			wantFailure: metrics.NewInt(20), // Not computed, original count.
		},
		{
			total:       metrics.NewInt(1000),
			success:     metrics.NewInt(1000),
			wantFailure: metrics.NewInt(0),
		},
		{
			success: metrics.NewInt(990),
		},
		{
			total: metrics.NewInt(1000),
		},
		{
			total:   metrics.NewInt(1000),
			success: metrics.NewString("100"),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%+v", test), func(t *testing.T) {
			em := metrics.NewEventMetrics(time.Now()).
				AddMetric("total", test.total).
				AddMetric("success", test.success)

			if test.failure != nil {
				em.AddMetric("failure", test.failure)
			}

			if err := AddFailureMetric(em); err != nil {
				if !test.wantErr {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Errorf("didn't get expected error")
			}

			gotFailure := em.Metric("failure")

			if test.wantFailure == nil {
				if gotFailure != nil {
					t.Errorf("Unexpected failure count metric (with value: %d)", gotFailure.(metrics.NumValue).Int64())
				}
				return
			}

			if test.wantFailure != nil && gotFailure == nil {
				t.Errorf("Not creating failure count metric; expected metric with value: %d", test.wantFailure.Int64())
				return
			}

			if gotFailure.(metrics.NumValue).Int64() != test.wantFailure.Int64() {
				t.Errorf("Failure count=%v, want=%v", test.wantFailure.Int64(), test.wantFailure.Int64())
			}
		})
	}
}
