// Copyright 2019 Google Inc.
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

package endpoint

import (
	"testing"
)

func TestEndpointsFromNames(t *testing.T) {
	names := []string{"targetA", "targetB", "targetC"}
	endpoints := EndpointsFromNames(names)

	for i := range names {
		ep := endpoints[i]

		if ep.Name != names[i] {
			t.Errorf("Endpoint.Name=%s, want=%s", ep.Name, names[i])
		}
		if ep.Port != 0 {
			t.Errorf("Endpoint.Port=%d, want=0", ep.Port)
		}
		if len(ep.Labels) != 0 {
			t.Errorf("Endpoint.Labels=%v, want={}", ep.Labels)
		}
	}
}
