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

package kubernetes

import (
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
