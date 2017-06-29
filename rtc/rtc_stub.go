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

package rtc

import (
	"encoding/base64"
	"errors"
	"fmt"

	"google.golang.org/api/runtimeconfig/v1beta1"
)

// Stub provides a stubbed version of RTC that can be used for testing larger
// components.
type Stub struct {
	m map[string]*runtimeconfig.Variable
}

// NewStub returns a Stub interface to RTC which will store all variables in an
// in-memory map.
func NewStub() *Stub {
	return &Stub{make(map[string]*runtimeconfig.Variable)}
}

// Write will add or change a key/val pair.
func (s *Stub) Write(key string, val []byte) error {
	encoded := base64.StdEncoding.EncodeToString(val)
	s.m[key] = &runtimeconfig.Variable{Name: key, Value: encoded}
	return nil
}

// WriteTime adds or changes a key/val pair, but with the UpdateTime field
// set to the given string. This is passed in as a string rather than a time so
// invalid times can be tested.
func (s *Stub) WriteTime(key string, val string, time string) error {
	s.m[key] = &runtimeconfig.Variable{Name: key, Value: val, UpdateTime: time}
	return nil
}

// Delete removes a key/val pair.
func (s *Stub) Delete(key string) error {
	_, ok := s.m[key]
	if ok {
		delete(s.m, key)
		return nil
	}
	return errors.New("rtc_stub: key not in map")
}

// Val gets the value associated with a variable.
func (s *Stub) Val(v *runtimeconfig.Variable) ([]byte, error) {
	v = s.m[v.Name]
	if v == nil {
		return nil, fmt.Errorf("rtc_stub: key %q not in map", v.Name)
	}
	return base64.StdEncoding.DecodeString(v.Value)
}

// List provides all vals in the map. To match the behavior of rtc.go, this
// will not include the value (to test that your code does not rely on such a
// behavior).
func (s *Stub) List() ([]*runtimeconfig.Variable, error) {
	vals := make([]*runtimeconfig.Variable, 0, len(s.m))
	for _, v := range s.m {
		vals = append(vals, &runtimeconfig.Variable{
			Name:       v.Name,
			UpdateTime: v.UpdateTime,
		})
	}
	return vals, nil
}

// FilterList is not supported by this Stub. Simply returns s.List().
func (s *Stub) FilterList(filter string) ([]*runtimeconfig.Variable, error) {
	return s.List()
}
