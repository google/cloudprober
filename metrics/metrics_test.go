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

package metrics

import (
	"testing"
)

func TestParseValueFromString(t *testing.T) {
	var val string

	// Bad value, should return an error
	val = "234g"
	_, err := ParseValueFromString(val)
	if err == nil {
		t.Errorf("ParseValueFromString(%s) returned no error", val)
	}

	// Float value
	val = "234"
	v, err := ParseValueFromString(val)
	if err != nil {
		t.Errorf("ParseValueFromString(%s) returned error: %v", val, err)
	}
	if _, ok := v.(*Float); !ok {
		t.Errorf("ParseValueFromString(%s) returned a non-float: %v", val, v)
	}

	// String value, aggregation disabled
	val = "\"234\""
	v, err = ParseValueFromString(val)
	if err != nil {
		t.Errorf("ParseValueFromString(%s) returned error: %v", val, err)
	}
	if _, ok := v.(String); !ok {
		t.Errorf("ParseValueFromString(%s) returned a non-string: %v", val, v)
	}

	// Map value
	val = "map:code 200:10 404:1"
	v, err = ParseValueFromString(val)
	if err != nil {
		t.Errorf("ParseValueFromString(%s) returned error: %v", val, err)
	}
	if _, ok := v.(*Map); !ok {
		t.Errorf("ParseValueFromString(%s) returned a non-map: %v", val, v)
	}

	// Dist value
	val = "dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12"
	v, err = ParseValueFromString(val)
	if err != nil {
		t.Errorf("ParseValueFromString(%s) returned error: %v", val, err)
	}
	if _, ok := v.(*Distribution); !ok {
		t.Errorf("ParseValueFromString(%s) returned a non-dist: %v", val, v)
	}
}
