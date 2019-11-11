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

import (
	"errors"
	"strconv"
	"strings"
)

// Value represents any metric value
type Value interface {
	Clone() Value
	Add(delta Value) error
	AddInt64(i int64)
	AddFloat64(f float64)
	String() string
}

// NumValue represents any numerical metric value, e.g. Int, Float.
// It's a superset of Value interface.
type NumValue interface {
	Value
	Inc()
	Int64() int64
	Float64() float64
	IncBy(delta NumValue)
}

// ParseValueFromString parses a value from its string representation
func ParseValueFromString(val string) (Value, error) {
	c := val[0]
	switch {
	// A float value
	case '0' <= c && c <= '9':
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, err
		}
		return NewFloat(f), nil

	// A map value
	case c == 'm':
		if !strings.HasPrefix(val, "map") {
			break
		}
		return ParseMapFromString(val)

	// A string value
	case c == '"':
		return NewString(strings.Trim(val, "\"")), nil

	// A distribution value
	case c == 'd':
		if !strings.HasPrefix(val, "dist") {
			break
		}
		distVal, err := ParseDistFromString(val)
		if err != nil {
			return nil, err
		}
		return distVal, nil
	}

	return nil, errors.New("unknown value type")
}
