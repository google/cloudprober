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
	"reflect"
	"testing"
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
