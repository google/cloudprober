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

// Package endpoint provides the type Endpoint, to be used with the
// targets.Targets interface.
package endpoint

// Endpoint represents a targets and associated parameters.
type Endpoint struct {
	Name   string
	Labels map[string]string
	Port   int
}

// EndpointsFromNames is convenience function to build a list of endpoints
// from only names. It leaves the Port field in Endpoint unset and initializes
// Labels field to an empty map.
func EndpointsFromNames(names []string) []Endpoint {
	result := make([]Endpoint, len(names))
	for i, name := range names {
		result[i].Name = name
		result[i].Labels = make(map[string]string)
	}
	return result
}
