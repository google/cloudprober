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

package cloudwatch

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/surfacers/cloudwatch/proto"
)

func newTestCWSurfacer() CWSurfacer {
	l, _ := logger.New(context.TODO(), "test-logger")
	namespace := "sre/test/cloudprober"
	resolution := int64(60)

	return CWSurfacer{
		l: l,
		c: &configpb.SurfacerConf{
			Namespace:           &namespace,
			AllowedMetricsRegex: new(string),
			Resolution:          &resolution,
		},
	}
}

func TestEmLabelsToDimensions(t *testing.T) {
	timestamp := time.Now()

	tests := map[string]struct {
		em   *metrics.EventMetrics
		want []*cloudwatch.Dimension
	}{
		"no label": {
			em:   metrics.NewEventMetrics(timestamp),
			want: []*cloudwatch.Dimension{},
		},
		"one label": {
			em: metrics.NewEventMetrics(timestamp).
				AddLabel("ptype", "sysvars"),
			want: []*cloudwatch.Dimension{
				{
					Name:  aws.String("ptype"),
					Value: aws.String("sysvars"),
				},
			},
		},
		"three labels": {
			em: metrics.NewEventMetrics(timestamp).
				AddLabel("ptype", "sysvars").
				AddLabel("probe", "sysvars").
				AddLabel("test", "testing123"),
			want: []*cloudwatch.Dimension{
				{
					Name:  aws.String("ptype"),
					Value: aws.String("sysvars"),
				},
				{
					Name:  aws.String("probe"),
					Value: aws.String("sysvars"),
				},
				{
					Name:  aws.String("test"),
					Value: aws.String("testing123"),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := emLabelsToDimensions(tc.em)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got: %v, want: %v", got, tc.want)
			}
		})
	}
}

func TestNewCWMetricDatum(t *testing.T) {
	timestamp := time.Now()

	tests := map[string]struct {
		surfacer   CWSurfacer
		metricname string
		value      float64
		dimensions []*cloudwatch.Dimension
		timestamp  time.Time
		duration   time.Duration
		want       *cloudwatch.MetricDatum
	}{
		"simple": {
			surfacer:   newTestCWSurfacer(),
			metricname: "testingmetric",
			value:      float64(20),
			dimensions: []*cloudwatch.Dimension{
				{
					Name: aws.String("test"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			want: &cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name: aws.String("test"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("testingmetric"),
				Value:             aws.Float64(float64(20)),
				StorageResolution: aws.Int64(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              aws.String(cloudwatch.StandardUnitCount),
			},
		},
		"le_dimension_count_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "testingmetric",
			value:      float64(20),
			dimensions: []*cloudwatch.Dimension{
				{
					Name: aws.String("test"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			want: &cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name: aws.String("test"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("testingmetric"),
				Value:             aws.Float64(float64(20)),
				StorageResolution: aws.Int64(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              aws.String("Count"),
			},
		},
		"latency_name_nanosecond_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "latency",
			value:      float64(20),
			dimensions: []*cloudwatch.Dimension{
				{
					Name: aws.String("name"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			duration:  time.Nanosecond,
			want: &cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name: aws.String("name"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(0.00002),
				StorageResolution: aws.Int64(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              aws.String("Milliseconds"),
			},
		},
		"latency_name_microseconds_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "latency",
			value:      float64(20),
			dimensions: []*cloudwatch.Dimension{
				{
					Name: aws.String("name"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			duration:  time.Microsecond,
			want: &cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name: aws.String("name"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(0.02),
				StorageResolution: aws.Int64(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              aws.String("Milliseconds"),
			},
		},
		"latency_name_milliseconds_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "latency",
			value:      float64(20),
			dimensions: []*cloudwatch.Dimension{
				{
					Name: aws.String("name"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			duration:  time.Millisecond,
			want: &cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name: aws.String("name"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(20),
				StorageResolution: aws.Int64(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              aws.String("Milliseconds"),
			},
		},
		"latency_name_seconds_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "latency",
			value:      float64(20),
			dimensions: []*cloudwatch.Dimension{
				{
					Name: aws.String("name"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			duration:  time.Second,
			want: &cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name: aws.String("name"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(20000),
				StorageResolution: aws.Int64(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              aws.String("Milliseconds"),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.surfacer.newCWMetricDatum(
				tc.metricname,
				tc.value,
				tc.dimensions,
				tc.timestamp,
				tc.duration,
			)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got: %v, want: %v", got, tc.want)
			}
		})
	}
}

func TestCalculateFailureMetric(t *testing.T) {
	timestamp := time.Now()

	tests := map[string]struct {
		em      *metrics.EventMetrics
		want    float64
		wantErr string
	}{
		"simple": {
			em: metrics.NewEventMetrics(timestamp).
				AddLabel("ptype", "http").
				AddMetric("total", metrics.NewFloat(10)).
				AddMetric("success", metrics.NewFloat(5)),
			want:    float64(5),
			wantErr: "",
		},
		"string metric": {
			em: metrics.NewEventMetrics(timestamp).
				AddLabel("ptype", "http").
				AddMetric("total", metrics.NewFloat(10)).
				AddMetric("success", metrics.NewString("5")),
			want:    float64(0),
			wantErr: "unexpected",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotFailure, err := calculateFailureMetric(tc.em)

			if !ErrorContains(err, tc.wantErr) {
				t.Errorf("unexpected error: %s", err)
			}

			if gotFailure != tc.want {
				t.Errorf("Return failure metric check: got: %f, want: %f", gotFailure, tc.want)
			}
		})
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
