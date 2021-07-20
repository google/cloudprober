// Copyright 2021 The Cloudprober Authors.
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

package datadog

import (
	"encoding/json"
	"io"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cAPIKey, cAppKey := "c-apiKey", "c-appKey"
	eAPIKey, eAppKey := "e-apiKey", "e-appKey"

	tests := []struct {
		desc       string
		apiKey     string
		appKey     string
		server     string
		env        map[string]string
		wantClient *ddClient
	}{
		{
			desc:   "keys-from-config",
			apiKey: cAPIKey,
			appKey: cAppKey,
			server: "",
			wantClient: &ddClient{
				apiKey: cAPIKey,
				appKey: cAppKey,
				server: defaultServer,
			},
		},
		{
			desc: "keys-from-env",
			env: map[string]string{
				"DD_API_KEY": eAPIKey,
				"DD_APP_KEY": eAppKey,
			},
			server: "test-server",
			wantClient: &ddClient{
				apiKey: eAPIKey,
				appKey: eAppKey,
				server: "test-server",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			for k, v := range test.env {
				os.Setenv(k, v)
			}

			c := newClient(test.server, test.apiKey, test.appKey)
			if !reflect.DeepEqual(c, test.wantClient) {
				t.Errorf("got client: %v, want client: %v", c, test.wantClient)
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	ts := time.Now().Unix()
	tags := []string{"probe:cloudprober_http"}
	metricType := "count"

	testSeries := []ddSeries{
		{
			Metric: "cloudprober.success",
			Points: [][]float64{[]float64{float64(ts), 99}},
			Tags:   &tags,
			Type:   &metricType,
		},
		{
			Metric: "cloudprober.total",
			Points: [][]float64{[]float64{float64(ts), 100}},
			Tags:   &tags,
			Type:   &metricType,
		},
	}

	testClient := newClient("", "test-api-key", "test-app-key")
	req, err := testClient.newRequest(testSeries)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check URL
	wantURL := "https://api.datadoghq.com/api/v1/series"
	if req.URL.String() != wantURL {
		t.Errorf("Got URL: %s, wanted: %s", req.URL.String(), wantURL)
	}

	// Check request headers
	for k, v := range map[string]string{
		"DD-API-KEY": "test-api-key",
		"DD-APP-KEY": "test-app-key",
	} {
		if req.Header.Get(k) != v {
			t.Errorf("%s header: %s, wanted: %s", k, req.Header.Get(k), v)
		}
	}

	// Check request body
	b, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Error reading request body: %v", err)
	}
	data := map[string][]ddSeries{}
	if err := json.Unmarshal(b, &data); err != nil {
		t.Errorf("Error unmarshaling request body: %v", err)
	}
	if !reflect.DeepEqual(data["series"], testSeries) {
		t.Errorf("s.Series: %v, testSeries: %v", data["series"], testSeries)
	}
}
