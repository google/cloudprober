// Copyright 2017-2018 The Cloudprober Authors.
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
Package filter implements common filters for the RDS (resource discovery
service) providers.
*/
package filter

import (
	"regexp"
	"time"

	"github.com/google/cloudprober/logger"
)

// RegexFilter implements a regex based filter.
type RegexFilter struct {
	re *regexp.Regexp
}

// NewRegexFilter returns a new regex filter.
func NewRegexFilter(regexStr string) (*RegexFilter, error) {
	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}

	return &RegexFilter{re}, nil
}

// Match returns true if provided string matches the regex of the filter.
// Otherwise, false is returned.
func (rf *RegexFilter) Match(name string, l *logger.Logger) bool {
	return rf.re.MatchString(name)
}

// LabelsFilter implements a filter on resource's labels.
type LabelsFilter struct {
	labels map[string]*regexp.Regexp
}

// NewLabelsFilter builds LabelsFilter from a key:regexp map.
func NewLabelsFilter(labelsFilter map[string]string) (*LabelsFilter, error) {
	labels := make(map[string]*regexp.Regexp)

	for key, regexStr := range labelsFilter {
		re, err := regexp.Compile(regexStr)
		if err != nil {
			return nil, err
		}
		labels[key] = re
	}

	return &LabelsFilter{labels: labels}, nil
}

// Match returns true if provided string matches the regex of the filter.
// Otherwise, false is returned.
func (lf *LabelsFilter) Match(inputLabels map[string]string, l *logger.Logger) bool {
	for k, re := range lf.labels {
		// If input labels don't have the requisite key or key's value doesn't match
		// the given regex, return false.
		if v, ok := inputLabels[k]; !ok || !re.MatchString(v) {
			return false
		}
	}
	return true
}

// FreshnessFilter implements a filter that succeeds only if the given time
// is within a pre-defined duration.
type FreshnessFilter struct {
	d time.Duration
}

// NewFreshnessFilter returns a new freshness filter.
func NewFreshnessFilter(dStr string) (*FreshnessFilter, error) {
	d, err := time.ParseDuration(dStr)
	return &FreshnessFilter{d}, err
}

// Match returns true if the given time is within a pre-defined duration.
func (ff *FreshnessFilter) Match(t time.Time, l *logger.Logger) bool {
	return time.Since(t) < ff.d
}
