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

package kubernetes

import (
	"encoding/json"
)

// resource represents a kubernetes resource, e.g. pod, service, etc.
type resource struct {
	name      string
	namespace string
	ip        string
	labels    map[string]string
}

// Temporary stuct to help parse JSON data.
type jsonItem struct {
	Metadata struct {
		Name      string
		Namespace string
		Labels    map[string]string
	}
	Spec   json.RawMessage
	Status json.RawMessage
}

// parseResourceList parses the JSON representation of a list of resources.
func parseResourceList(jsonResp []byte, ipFunc func(jsonItem) (string, error)) ([]string, map[string]*resource, error) {
	var itemList struct {
		Items []jsonItem
	}

	if err := json.Unmarshal(jsonResp, &itemList); err != nil {
		return nil, nil, err
	}

	names := make([]string, len(itemList.Items))
	resources := make(map[string]*resource, len(names))

	for i, item := range itemList.Items {
		r := &resource{
			name:      item.Metadata.Name,
			namespace: item.Metadata.Namespace,
			labels:    item.Metadata.Labels,
		}
		if ipFunc != nil {
			ip, err := ipFunc(item)
			if err != nil {
				return nil, nil, err
			}
			r.ip = ip
		}
		names[i] = r.name
		resources[r.name] = r
	}

	return names, resources, nil
}
