package cloudwatch

import (
	"context"
	"reflect"
	"regexp"
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
		l:                 l,
		ignoreLabelsRegex: &regexp.Regexp{},
		c: &configpb.SurfacerConf{
			Namespace:  &namespace,
			Resolution: &resolution,
		},
	}
}

func TestIgnoreProberTypeLabel(t *testing.T) {
	timestamp := time.Now()

	tests := map[string]struct {
		surfacer     CWSurfacer
		em           *metrics.EventMetrics
		labels       map[string]string
		regexpString string
		want         bool
	}{
		"regexp match": {
			surfacer: newTestCWSurfacer(),
			em:       metrics.NewEventMetrics(timestamp),
			labels: map[string]string{
				"ptype": "sysvars",
				"probe": "testprobe",
			},
			regexpString: "sysvars",
			want:         true,
		},
		"regexp miss": {
			surfacer: newTestCWSurfacer(),
			em:       metrics.NewEventMetrics(timestamp),
			labels: map[string]string{
				"ptype": "http",
				"probe": "testprobe",
			},
			regexpString: "sysvars",
			want:         false,
		},
		"regexp wildcard all": {
			surfacer: newTestCWSurfacer(),
			em:       metrics.NewEventMetrics(timestamp),
			labels: map[string]string{
				"ptype": "sysvars",
				"probe": "testprobe",
			},
			regexpString: ".*",
			want:         true,
		},
		"regexp partial match": {
			surfacer: newTestCWSurfacer(),
			em:       metrics.NewEventMetrics(timestamp),
			labels: map[string]string{
				"ptype": "sysvars",
				"probe": "testprobe",
			},
			regexpString: "sys",
			want:         true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.labels {
				tc.em.AddLabel(k, v)
			}

			r, err := regexp.Compile(tc.regexpString)
			if err != nil {
				t.Fatalf("Error compiling regex string: %s, error: %s", tc.regexpString, err)
			}

			tc.surfacer.ignoreLabelsRegex = r
			got := tc.surfacer.ignoreProberTypeLabel(tc.em)
			if got != tc.want {
				t.Errorf("got: %t, want: %t", got, tc.want)
			}
		})
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
			},
		},
		"le_dimension_count_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "testingmetric",
			value:      float64(20),
			dimensions: []*cloudwatch.Dimension{
				{
					Name: aws.String("le"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			want: &cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name: aws.String("le"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("testingmetric"),
				Value:             aws.Float64(float64(20)),
				StorageResolution: aws.Int64(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              aws.String("Count"),
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
			want: &cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name: aws.String("name"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(float64(20)),
				StorageResolution: aws.Int64(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              aws.String("Milliseconds"),
			},
		},
		"latency_name_but_le_dimension_count_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "latency",
			value:      float64(20),
			dimensions: []*cloudwatch.Dimension{
				{
					Name: aws.String("le"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			want: &cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					{
						Name: aws.String("le"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(float64(20)),
				StorageResolution: aws.Int64(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              aws.String("Count"),
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
