// Copyright 2019 The Cloudprober Authors.
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
	"fmt"
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

func TestKey(t *testing.T) {
	for _, test := range []struct {
		name   string
		port   int
		labels map[string]string
		key    string
	}{
		{
			name: "t1",
			port: 80,
			key:  "t1_80",
		},
		{
			name:   "t1",
			port:   80,
			labels: map[string]string{"app": "cloudprober", "dc": "xx"},
			key:    "t1_80_app:cloudprober_dc:xx",
		},
		{
			name:   "t1",
			port:   80,
			labels: map[string]string{"dc": "xx", "app": "cloudprober"},
			key:    "t1_80_app:cloudprober_dc:xx",
		},
	} {
		ep := Endpoint{
			Name:   test.name,
			Port:   test.port,
			Labels: test.labels,
		}
		t.Run(fmt.Sprintf("%v", ep), func(t *testing.T) {
			key := ep.Key()
			if key != test.key {
				t.Errorf("Got key: %s, want: %s", key, test.key)
			}
		})
	}
}
