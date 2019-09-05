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
	"io/ioutil"
	"testing"
)

func TestHTTPRequest(t *testing.T) {
	c := &client{
		bearer:  "testtoken",
		apiHost: "testHost",
	}

	testURL := "/test-url"

	req, err := c.httpRequest(testURL)

	if err != nil {
		t.Errorf("Unexpected error while creating HTTP request from URL (%s): %v", testURL, err)
	}

	if req.Host != c.apiHost {
		t.Errorf("Got host = %s, expected = %s", req.Host, c.apiHost)
	}

	if req.URL.Path != testURL {
		t.Errorf("Got URL path = %s, expected = %s", req.URL.Path, testURL)
	}

	if req.Header.Get("Authorization") != c.bearer {
		t.Errorf("Got Authorization Header = %s, expected = %s", req.Header.Get("Authorization"), c.bearer)
	}
}

func TestParseResourceList(t *testing.T) {
	podsListFile := "./testdata/pods.json"
	data, err := ioutil.ReadFile(podsListFile)

	if err != nil {
		t.Fatalf("error reading test data file: %s", podsListFile)
	}

	_, podsByName, err := parseResourceList(data, func(item jsonItem) (string, error) { return podIPFunc(item, nil) })
	if err != nil {
		t.Errorf("error parsing test json data: %s", string(data))
	}

	cpPod := "cloudprober-54778d95f5-7hqtd"
	testNames := []string{cpPod, "test"}
	for _, testP := range testNames {
		if podsByName[testP] == nil {
			t.Errorf("didn't get pod by the name: %s", testP)
		}
	}

	if podsByName[cpPod].labels["app"] != "cloudprober" {
		t.Errorf("cloudprober pod app label: got=%s, want=cloudprober", podsByName[cpPod].labels["app"])
	}

	cpPodIP := "10.28.0.3"
	if podsByName[cpPod].ip != cpPodIP {
		t.Errorf("cloudprober pod ip: got=%s, want=%s", podsByName[cpPod].ip, cpPodIP)
	}
}
