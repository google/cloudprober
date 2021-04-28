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

// Package endpoint provides the type Endpoint, to be used with the
// targets.Targets interface.
package endpoint

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Endpoint represents a targets and associated parameters.
type Endpoint struct {
	Name        string
	Labels      map[string]string
	LastUpdated time.Time
	Port        int
}

// Key returns a string key that uniquely identifies that endpoint.
// Endpoint key consists of endpoint name, port and labels.
func (ep *Endpoint) Key() string {
	labelSlice := make([]string, len(ep.Labels))
	i := 0
	for k, v := range ep.Labels {
		labelSlice[i] = k + ":" + v
		i++
	}
	sort.Strings(labelSlice)

	return strings.Join(append([]string{ep.Name, strconv.Itoa(ep.Port)}, labelSlice...), "_")
}

// Lister should implement the ListEndpoints method.
type Lister interface {
	// ListEndpoints returns list of endpoints (name, port tupples).
	ListEndpoints() []Endpoint
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

// NamesFromEndpoints is convenience function to build a list of names
// from endpoints.
func NamesFromEndpoints(endpoints []Endpoint) []string {
	result := make([]string, len(endpoints))
	for i, ep := range endpoints {
		result[i] = ep.Name
	}
	return result
}
