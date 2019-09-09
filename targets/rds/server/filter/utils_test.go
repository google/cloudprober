package filter

import (
	"reflect"
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/google/cloudprober/targets/rds/proto"
)

func TestParseFilters(t *testing.T) {
	tests := []struct {
		desc                string
		reqFilters          map[string]string
		regexFilterKeys     []string
		freshnessFilterKey  string
		wantReFilters       []string
		wantLabelsFilter    bool
		wantFreshnessFilter bool
		wantErr             bool
	}{
		{
			desc:            "Error invalid filter key",
			reqFilters:      map[string]string{"random_key": "random_value"},
			regexFilterKeys: []string{"name"},
			wantErr:         true,
		},
		{
			desc: "Pass with 2 regex filters, labels filter and a freshness filter",
			reqFilters: map[string]string{
				"name":           "x.*",
				"namespace":      "y",
				"labels.app":     "cloudprober",
				"updated_within": "5m",
			},
			regexFilterKeys:     []string{"name", "namespace"},
			freshnessFilterKey:  "updated_within",
			wantReFilters:       []string{"name", "namespace"},
			wantLabelsFilter:    true,
			wantFreshnessFilter: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var reqFiltersPB []*pb.Filter
			for k, v := range test.reqFilters {
				reqFiltersPB = append(reqFiltersPB, &pb.Filter{Key: proto.String(k), Value: proto.String(v)})
			}

			allFilters, err := ParseFilters(reqFiltersPB, test.regexFilterKeys, test.freshnessFilterKey)

			if err != nil {
				if !test.wantErr {
					t.Errorf("Got unexpected error while parsing filters: %v", err)
				}
				return
			}

			var reFilterKeys []string
			for k := range allFilters.RegexFilters {
				reFilterKeys = append(reFilterKeys, k)
			}

			sort.Strings(reFilterKeys)
			sort.Strings(test.wantReFilters)
			if !reflect.DeepEqual(reFilterKeys, test.wantReFilters) {
				t.Errorf("regex filters, got=%v, want=%v", reFilterKeys, test.wantReFilters)
			}

			if (allFilters.LabelsFilter != nil) != test.wantLabelsFilter {
				t.Errorf("labels filters, got=%v, wantLabelsFilter=%v", allFilters.LabelsFilter, test.wantLabelsFilter)
			}

			if (allFilters.FreshnessFilter != nil) != test.wantFreshnessFilter {
				t.Errorf("freshness filter filters, got=%v, wantFreshnessFilter=%v", allFilters.FreshnessFilter, test.wantFreshnessFilter)
			}
		})
	}

}
