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

package http

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

func TestListenAndServeInstanceURL(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Listen error: %v.", err)
	}

	testSysVars := map[string]string{
		"instance": "testInstance",
	}
	dataChan := make(chan *metrics.EventMetrics, 10)
	go func() {
		t.Fatal(serve(context.Background(), ln, dataChan, testSysVars, &logger.Logger{}))
	}()
	resp, err := http.Get(fmt.Sprintf("http://%s/instance", ln.Addr().String()))
	if err != nil {
		t.Errorf("HTTP server returned an error for the URL '/instance'. Err: %v", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error while reading response for the URL '/instance': Err: %v", err)
		return
	}
	if string(body) != testSysVars["instance"] {
		t.Errorf("Didn't get the expected response for the URL '/instance'. Got: %s, Expected: %s", string(body), testSysVars["instance"])
	}
}

func TestListenAndServeStats(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Listen error: %v.", err)
	}
	dataChan := make(chan *metrics.EventMetrics, 10)
	testURLs := []string{"/", "/testURL1", "/", "/testURL2"}
	// Set global variable statsExportInterval to a lower value to shorten the test
	// run time.
	statsExportInterval = 2 * time.Second

	go func() {
		t.Fatal(serve(context.Background(), ln, dataChan, make(map[string]string), &logger.Logger{}))
	}()
	for _, url := range testURLs {
		resp, err := http.Get(fmt.Sprintf("http://%s/%s", ln.Addr().String(), url))
		if err != nil {
			t.Errorf("HTTP server returned an error for URL '%s'. Err: %v", url, err)
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Error while reading response for URL '%s': %v", url, err)
			continue
		}
		if string(body) != defaultResponse {
			t.Errorf("Didn't get the expected response for URL '%s'. Got: %s, Expected: %s", url, string(body), defaultResponse)
		}
	}
	// Sleep for the export interval and a second extra to allow for the stats to
	// come in before closing the channel.
	time.Sleep(statsExportInterval)
	time.Sleep(time.Second)
	close(dataChan)

	// Build a map of expected URL stats
	expectedURLStats := make(map[string]int64)
	for _, url := range testURLs {
		expectedURLStats[url]++
	}
	if len(dataChan) != 1 {
		t.Errorf("Wrong number of stats on the stats channel. Got: %d, Expected: %d", len(dataChan), 1)
	}
	em := <-dataChan

	// See if we got stats for the all URLs
	for url, expectedCount := range expectedURLStats {
		count := em.Metric("req").(*metrics.Map).GetKey(url).Int64()
		if count != expectedCount {
			t.Errorf("Didn't get the expected stats for the URL: %s. Got: %d, Expected: %d", url, count, expectedCount)
		}
	}
}
