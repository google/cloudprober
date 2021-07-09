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

package stackdriver

import (
	"context"
	"testing"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/kylelemons/godebug/pretty"
	monitoring "google.golang.org/api/monitoring/v3"
)

var (
	stringVal = "string-test"
	boolVal   = true
	floatVal  = 123.456
	intVal    = float64(123456)
)

func newTestSurfacer() SDSurfacer {
	l, _ := logger.New(context.TODO(), "test-logger")
	return SDSurfacer{
		cache:       make(map[string]*monitoring.TimeSeries),
		onGCE:       true,
		projectName: "test-project",
		l:           l,
		resource: &monitoring.MonitoredResource{
			Type: "gce_instance",
			Labels: map[string]string{
				"instance_id": "test-instance",
				"zone":        "us-central1-a",
			},
		},
	}
}

func TestTimeSeries(t *testing.T) {
	testTimestamp := time.Now()

	oneVal := float64(1)

	// Following variables are used for map value testing.
	mapValue200 := float64(98)
	mapValue500 := float64(2)
	mapVal := metrics.NewMap("code", metrics.NewInt(0))
	mapVal.IncKeyBy("200", metrics.NewInt(int64(mapValue200)))
	mapVal.IncKeyBy("500", metrics.NewInt(int64(mapValue500)))

	tests := []struct {
		description string
		surfacer    SDSurfacer
		em          *metrics.EventMetrics
		timeSeries  []*monitoring.TimeSeries
	}{
		{
			description: "timeseries creation with a non-default float64 value",
			surfacer:    newTestSurfacer(),
			em:          metrics.NewEventMetrics(testTimestamp).AddMetric("test-message", metrics.NewInt(123456)),
			timeSeries: []*monitoring.TimeSeries{
				&monitoring.TimeSeries{
					Metric: &monitoring.Metric{
						Type: "custom.googleapis.com/cloudprober/test-message",
					},
					Resource: &monitoring.MonitoredResource{
						Type: "gce_instance",
						Labels: map[string]string{
							"instance_id": "test-instance",
							"zone":        "us-central1-a",
						},
					},
					MetricKind: "CUMULATIVE",
					ValueType:  "DOUBLE",
					Unit:       "1",
					Points: []*monitoring.Point{
						{
							Interval: &monitoring.TimeInterval{
								StartTime: "0001-01-01T00:00:00Z",
								EndTime:   testTimestamp.Format(time.RFC3339Nano),
							},
							Value: &monitoring.TypedValue{
								DoubleValue: &intVal,
							},
						},
					},
				},
			},
		},
		{
			description: "timeseries creation with a non-default string value and labels",
			surfacer:    newTestSurfacer(),
			em: metrics.NewEventMetrics(testTimestamp).
				AddMetric("version", metrics.NewString("versionXX")).
				AddLabel("keyA", "valueA").
				AddLabel("keyB", "valueB"),
			timeSeries: []*monitoring.TimeSeries{
				&monitoring.TimeSeries{
					Metric: &monitoring.Metric{
						Type: "custom.googleapis.com/cloudprober/version",
						Labels: map[string]string{
							"keyA": "valueA",
							"keyB": "valueB",
							"val":  "versionXX",
						},
					},
					Resource: &monitoring.MonitoredResource{
						Type: "gce_instance",
						Labels: map[string]string{
							"instance_id": "test-instance",
							"zone":        "us-central1-a",
						},
					},
					MetricKind: "CUMULATIVE",
					ValueType:  "DOUBLE",
					Unit:       "1",
					Points: []*monitoring.Point{
						{
							Interval: &monitoring.TimeInterval{
								StartTime: "0001-01-01T00:00:00Z",
								EndTime:   testTimestamp.Format(time.RFC3339Nano),
							},
							Value: &monitoring.TypedValue{
								DoubleValue: &oneVal,
							},
						},
					},
				},
			},
		},
		{
			description: "timeseries creation with a non-default map value and labels",
			surfacer:    newTestSurfacer(),
			em: metrics.NewEventMetrics(testTimestamp).
				AddMetric("version", mapVal).
				AddLabel("keyA", "valueA").
				AddLabel("keyB", "valueB"),
			timeSeries: []*monitoring.TimeSeries{
				&monitoring.TimeSeries{
					Metric: &monitoring.Metric{
						Type: "custom.googleapis.com/cloudprober/version",
						Labels: map[string]string{
							"keyA": "valueA",
							"keyB": "valueB",
							"code": "200",
						},
					},
					Resource: &monitoring.MonitoredResource{
						Type: "gce_instance",
						Labels: map[string]string{
							"instance_id": "test-instance",
							"zone":        "us-central1-a",
						},
					},
					MetricKind: "CUMULATIVE",
					ValueType:  "DOUBLE",
					Unit:       "1",
					Points: []*monitoring.Point{
						{
							Interval: &monitoring.TimeInterval{
								StartTime: "0001-01-01T00:00:00Z",
								EndTime:   testTimestamp.Format(time.RFC3339Nano),
							},
							Value: &monitoring.TypedValue{
								DoubleValue: &mapValue200,
							},
						},
					},
				},
				&monitoring.TimeSeries{
					Metric: &monitoring.Metric{
						Type: "custom.googleapis.com/cloudprober/version",
						Labels: map[string]string{
							"keyA": "valueA",
							"keyB": "valueB",
							"code": "500",
						},
					},
					Resource: &monitoring.MonitoredResource{
						Type: "gce_instance",
						Labels: map[string]string{
							"instance_id": "test-instance",
							"zone":        "us-central1-a",
						},
					},
					MetricKind: "CUMULATIVE",
					ValueType:  "DOUBLE",
					Unit:       "1",
					Points: []*monitoring.Point{
						{
							Interval: &monitoring.TimeInterval{
								StartTime: "0001-01-01T00:00:00Z",
								EndTime:   testTimestamp.Format(time.RFC3339Nano),
							},
							Value: &monitoring.TypedValue{
								DoubleValue: &mapValue500,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		// Generate a time series and check that it is correct
		gotTimeSeries := tt.surfacer.recordEventMetrics(tt.em)
		if diff := pretty.Compare(tt.timeSeries, gotTimeSeries); diff != "" {
			t.Errorf("timeSeries() produced incorrect timeSeries (-want +got):\n%s\ntest description: %s", diff, tt.description)
		}
	}
}
