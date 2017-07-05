// Copyright 2017 Google Inc.
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

package metrics

import (
	"reflect"
	"testing"
)

func verify(t *testing.T, m *Map, expectedKeys []string, expectedMap map[string]int64) {
	if !reflect.DeepEqual(m.Keys(), expectedKeys) {
		t.Errorf("Map doesn't have expected keys. Got: %q, Expected: %q", m.Keys(), expectedKeys)
	}
	for k, v := range expectedMap {
		if m.GetKey(k).Int64() != v {
			t.Errorf("Key values not as expected. Key: %s, Got: %d, Expected: %d", k, m.GetKey(k).Int64(), v)
		}
	}
}

func TestMap(t *testing.T) {
	m := NewMap("code", NewInt(0))
	m.IncKeyBy("200", NewInt(4000))

	verify(t, m, []string{"200"}, map[string]int64{"200": 4000})

	m.IncKey("500")
	verify(t, m, []string{"200", "500"}, map[string]int64{
		"200": 4000,
		"500": 1,
	})

	// Verify that keys are ordered
	m.IncKey("404")
	verify(t, m, []string{"200", "404", "500"}, map[string]int64{
		"200": 4000,
		"404": 1,
		"500": 1,
	})

	// Clone m for verification later
	m1 := m.clone().(*Map)

	// Verify add works as expected
	m2 := NewMap("code", NewInt(0))
	m2.IncKeyBy("403", NewInt(2))
	err := m.Add(m2)
	if err != nil {
		t.Errorf("Add two maps produced error. Err: %v", err)
	}
	verify(t, m, []string{"200", "403", "404", "500"}, map[string]int64{
		"200": 4000,
		"403": 2,
		"404": 1,
		"500": 1,
	})

	// Verify that clones value has not changed
	verify(t, m1, []string{"200", "404", "500"}, map[string]int64{
		"200": 4000,
		"404": 1,
		"500": 1,
	})
}
