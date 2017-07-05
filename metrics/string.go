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

// Package metrics implements data types for probes generated data.
package metrics

import "errors"

// String implements a value type with string storage.
// It satisfies the Value interface.
type String struct {
	s string
}

// NewString returns a new String with the given string value.
func NewString(s string) String {
	return String{s: s}
}

// Add isn't supported for String type, this is only to satisfy the Value interface.
func (s String) Add(val Value) error {
	return errors.New("string value type doesn't support Add() operation")
}

// String simply returns the stored string.
func (s String) String() string {
	return "\"" + s.s + "\""
}

func (s String) clone() Value {
	return String{s: s.s}
}
