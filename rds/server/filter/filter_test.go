// Copyright 2017-2019 The Cloudprober Authors.
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

package filter

import (
	"testing"
)

func TestLabelsFilter(t *testing.T) {
	for _, testData := range []struct {
		testLabels   []map[string]string
		labelFilters map[string]string
		expectError  bool
		matchCount   int
	}{
		{
			testLabels: []map[string]string{
				{"lA": "vAA", "lB": "vB"},
				{"lA": "vAB", "lB": "vB"},
			},
			labelFilters: map[string]string{"lA": "vA.", "lB": "vB"},
			expectError:  false,
			matchCount:   2,
		},
		{
			testLabels: []map[string]string{
				{"lA": "vAA", "lB": "vB"}, // Only this will match.
				{"lA": "vBB", "lB": "vB"},
			},
			labelFilters: map[string]string{"lA": "vA.", "lB": "vB"},
			expectError:  false,
			matchCount:   1,
		},
		{
			// Label lC not on any instance, no match.
			testLabels: []map[string]string{
				{"lA": "vAA", "lB": "vB"},
				{"lA": "vBB", "lB": "vB"},
			},
			labelFilters: map[string]string{"lC": "vC.", "lB": "vB"},
			expectError:  false,
			matchCount:   0,
		},
	} {
		lf, err := NewLabelsFilter(testData.labelFilters)
		if err != nil {
			t.Errorf("Got unexpected error while adding a label filter: %v", err)
		}

		var testMatchCount int
		for _, testInstance := range testData.testLabels {
			t.Logf("Matching instance with labels: %v", testInstance)
			if lf.Match(testInstance, nil) {
				testMatchCount++
			}
		}

		if testMatchCount != testData.matchCount {
			t.Errorf("Match count doesn't match with expected match count. Got=%d, Expected=%d", testMatchCount, testData.matchCount)
		}
	}
}
