package filter

import (
	"fmt"
	"strings"

	pb "github.com/google/cloudprober/targets/rds/proto"
)

// Filters encapsulates all types of filters. Its main purpose is to serve as a
// return type for ParseFilters.
type Filters struct {
	RegexFilters map[string]*RegexFilter
	*LabelsFilter
	*FreshnessFilter
}

// ParseFilters parses filter protobufs into Filters struct. Filters are parsed
// based on the following criteria:
//  - There can be multiple regex filters. Keys for these filters should be
//    provided through the regexFilterKeys argument.
//  - Labels filter keys always starts with the prefix "labels.".
//  - There can be only one freshness filter, key for which should be provided
//    through the freshnessFilterKey argument.
func ParseFilters(filters []*pb.Filter, regexFilterKeys []string, freshnessFilterKey string) (*Filters, error) {
	r := &Filters{
		RegexFilters: make(map[string]*RegexFilter),
	}

	// Initialize r.RegexFilters with expected regex filter keys for quick lookup.
	for _, k := range regexFilterKeys {
		r.RegexFilters[k] = nil
	}

	labels := make(map[string]string)

	for _, f := range filters {

		// If we expect this filter to be a regex filter.
		if _, ok := r.RegexFilters[f.GetKey()]; ok {
			rf, err := NewRegexFilter(f.GetValue())
			if err != nil {
				return nil, fmt.Errorf("filter: error creating regex filter from: %s=%s, err: %v", f.GetKey(), f.GetValue(), err)
			}
			r.RegexFilters[f.GetKey()] = rf
			continue
		}

		// If we expect this filter to be a freshness filter.
		if f.GetKey() == freshnessFilterKey {
			ff, err := NewFreshnessFilter(f.GetValue())
			if err != nil {
				return nil, fmt.Errorf("filter: error creating freshness filter from: %s=%s, err: %v", f.GetKey(), f.GetValue(), err)
			}
			r.FreshnessFilter = ff
			continue
		}

		// If it is a labels filter (starting with labels.).
		// Note: labels.<key> format matches with gcloud's filter options.
		if strings.HasPrefix(f.GetKey(), "labels.") {
			labels[strings.TrimPrefix(f.GetKey(), "labels.")] = f.GetValue()
			continue
		}

		// Unexpected filter key.
		return nil, fmt.Errorf("unsupported filter key: %s", f.GetKey())
	}

	if len(labels) != 0 {
		var err error
		if r.LabelsFilter, err = NewLabelsFilter(labels); err != nil {
			return nil, fmt.Errorf("filter: error creating labels filter from: %v, err: %v", labels, err)
		}
	}

	return r, nil
}
