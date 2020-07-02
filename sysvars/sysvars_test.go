// Copyright 2020 Google Inc.
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

package sysvars

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/cloudprober/logger"
)

func TestProvidersToCheck(t *testing.T) {
	flagToProviders := map[string][]string{
		"auto": []string{"gce", "ec2"},
		"gce":  []string{"gce"},
		"ec2":  []string{"ec2"},
		"none": nil,
	}

	for flagValue, expected := range flagToProviders {
		t.Run("testing_with_cloud_metadata="+flagValue, func(t *testing.T) {
			got := providersToCheck(flagValue)
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("providersToCheck(%s)=%v, expected=%v", flagValue, got, expected)
			}
		})
	}
}

var testGCEVars = map[string]string{
	"platform": "gce",
	"zone":     "gce-zone-1",
}

var testEC2Vars = map[string]string{
	"platform": "ec2",
	"zone":     "ec2-zone-1",
}

func testSetVars(vars, inVars map[string]string, onPlatform bool) (bool, error) {
	if !onPlatform {
		return onPlatform, nil
	}
	for k, v := range inVars {
		vars[k] = v
	}
	return onPlatform, nil
}

func TestInitCloudMetadata(t *testing.T) {
	sysVars = map[string]string{}

	tests := []struct {
		mode         string
		onGCE, onEC2 bool
		expected     map[string]string
	}{
		{
			mode:     "auto",
			onGCE:    true,
			onEC2:    true,
			expected: testGCEVars,
		},
		{
			mode:     "auto",
			onGCE:    false,
			onEC2:    true,
			expected: testEC2Vars,
		},
		{
			mode:     "gce",
			onGCE:    false,
			onEC2:    true, // running on EC2, got nothing as looking for GCE
			expected: map[string]string{},
		},
		{
			mode:     "ec2",
			onGCE:    true, // Running on GCE, got nothing as looking for EC2
			onEC2:    false,
			expected: map[string]string{},
		},
		{
			mode:     "ec2", // Get EC2 metadata
			onGCE:    true,
			onEC2:    true,
			expected: testEC2Vars,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {
			sysVars = map[string]string{}

			gceVars = func(vars map[string]string) (bool, error) {
				return testSetVars(vars, testGCEVars, test.onGCE)
			}
			ec2Vars = func(vars map[string]string, l *logger.Logger) (bool, error) {
				return testSetVars(vars, testEC2Vars, test.onEC2)
			}

			if err := initCloudMetadata(test.mode); err != nil {
				t.Errorf("Got unexpected error: %v", err)
			}
			if !reflect.DeepEqual(sysVars, test.expected) {
				t.Errorf("sysVars=%v, expected=%v", sysVars, test.expected)
			}
		})
	}
}
