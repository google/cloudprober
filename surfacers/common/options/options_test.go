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

package options

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/surfacers/proto"
	"google.golang.org/protobuf/proto"
)

var testEventMetrics = []*metrics.EventMetrics{
	metrics.NewEventMetrics(time.Now()).
		AddMetric("total", metrics.NewInt(20)).
		AddMetric("timeout", metrics.NewInt(2)).
		AddLabel("ptype", "http").
		AddLabel("probe", "manugarg_homepage"),
	metrics.NewEventMetrics(time.Now()).
		AddMetric("total", metrics.NewInt(20)).
		AddMetric("timeout", metrics.NewInt(2)).
		AddLabel("ptype", "http").
		AddLabel("probe", "google_homepage"),
	metrics.NewEventMetrics(time.Now()).
		AddMetric("memory", metrics.NewInt(20)).
		AddMetric("num_goroutines", metrics.NewInt(2)).
		AddLabel("probe", "sysvars"),
}

func TestAllowEventMetrics(t *testing.T) {
	tests := []struct {
		desc         string
		allowFilter  [][2]string
		ignoreFilter [][2]string
		wantAllowed  []int
		wantErr      bool
	}{
		{
			desc:        "all",
			wantAllowed: []int{0, 1, 2},
		},
		{
			desc: "ignore-sysvars-and-google-homepage",
			ignoreFilter: [][2]string{
				{"probe", "sysvars"},
				{"probe", "google_homepage"},
			},
			wantAllowed: []int{0},
		},
		{
			desc: "allow-google-homepage-and-sysvars",
			allowFilter: [][2]string{
				{"probe", "sysvars"},
				{"probe", "google_homepage"},
			},
			wantAllowed: []int{1, 2},
		},
		{
			desc: "ignore-takes-precedence-for-sysvars",
			allowFilter: [][2]string{
				{"probe", "sysvars"},
				{"probe", "google_homepage"},
			},
			ignoreFilter: [][2]string{
				{"probe", "sysvars"},
			},
			wantAllowed: []int{1},
		},
		{
			desc:        "error-label-value-without-key",
			allowFilter: [][2]string{{"", "sysvars"}},
			wantErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &configpb.SurfacerDef{}
			for _, ignoreF := range test.ignoreFilter {
				config.IgnoreMetricsWithLabel = append(config.IgnoreMetricsWithLabel, &configpb.LabelFilter{Key: proto.String(ignoreF[0]), Value: proto.String(ignoreF[1])})
			}
			for _, allowF := range test.allowFilter {
				config.AllowMetricsWithLabel = append(config.AllowMetricsWithLabel, &configpb.LabelFilter{Key: proto.String(allowF[0]), Value: proto.String(allowF[1])})
			}

			opts, err := BuildOptionsFromConfig(config)
			if err != nil {
				if !test.wantErr {
					t.Errorf("Unexpected building options from the config: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Errorf("Expected error, but there were none")
				return
			}

			var gotEM []int
			for i, em := range testEventMetrics {
				if opts.AllowEventMetrics(em) {
					gotEM = append(gotEM, i)
				}
			}

			if !reflect.DeepEqual(gotEM, test.wantAllowed) {
				t.Errorf("Got EMs (index): %v, want EMs (index): %v", gotEM, test.wantAllowed)
			}
		})
	}
}

func TestAllowMetric(t *testing.T) {
	tests := []struct {
		desc        string
		metricName  []string
		allow       string
		ignore      string
		wantMetrics []string
		wantErr     bool
	}{
		{
			desc:        "all",
			metricName:  []string{"total", "success"},
			wantMetrics: []string{"total", "success"},
		},
		{
			desc:       "bad-allow-regex",
			metricName: []string{"total", "success"},
			allow:      "(?badRe)",
			wantErr:    true,
		},
		{
			desc:       "bad-ignore-regex",
			metricName: []string{"total", "success"},
			ignore:     "(?badRe)",
			wantErr:    true,
		},
		{
			desc:        "ignore-total",
			metricName:  []string{"total", "success"},
			ignore:      "tot.*",
			wantMetrics: []string{"success"},
		},
		{
			desc:        "allow-total",
			metricName:  []string{"total", "success"},
			allow:       "tot.*",
			wantMetrics: []string{"total"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &configpb.SurfacerDef{
				IgnoreMetricsWithName: proto.String(test.ignore),
				AllowMetricsWithName:  proto.String(test.allow),
			}

			opts, err := BuildOptionsFromConfig(config)
			if err != nil {
				if !test.wantErr {
					t.Errorf("Unexpected building options from the config: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Errorf("Expected error, but there were none")
				return
			}

			var gotMetrics []string
			for _, m := range test.metricName {
				if opts.AllowMetric(m) {
					gotMetrics = append(gotMetrics, m)
				}
			}

			if !reflect.DeepEqual(gotMetrics, test.wantMetrics) {
				t.Errorf("Got metrics: %v, wanted: %v", gotMetrics, test.wantMetrics)
			}
		})
	}
}
