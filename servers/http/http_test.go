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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/targets/endpoint"
)

const testExportInterval = 2 * time.Second

type fakeLameduckLister struct {
	lameducked []string
	err        error
}

func (f *fakeLameduckLister) ListEndpoints() []endpoint.Endpoint {
	return endpoint.EndpointsFromNames(f.lameducked)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func testServer(ctx context.Context, t *testing.T, insName string, ldLister endpoint.Lister) (*Server, chan *metrics.EventMetrics) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Listen error: %v.", err)
	}

	dataChan := make(chan *metrics.EventMetrics, 10)
	s := &Server{
		l:             &logger.Logger{},
		ln:            ln,
		statsInterval: 2 * time.Second,
		instanceName:  insName,
		ldLister:      ldLister,
		reqMetric:     metrics.NewMap("url", metrics.NewInt(0)),
		staticURLResTable: map[string][]byte{
			"/":         []byte(OK),
			"/instance": []byte(insName),
		},
	}

	go func() {
		s.Start(ctx, dataChan)
	}()

	return s, dataChan
}

// get preforms HTTP GET request and return the response body and status
func get(t *testing.T, ln net.Listener, path string) (string, string) {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://%s/%s", listenerAddr(ln), path))
	if err != nil {
		t.Errorf("HTTP server returned an error for the URL '/%s'. Err: %v", path, err)
		return "", ""
	}
	status := resp.Status
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error while reading response for the URL '/%s': Err: %v", path, err)
		return "", status
	}
	return string(body), status
}

func listenerAddr(ln net.Listener) string {
	return fmt.Sprintf("localhost:%d", ln.Addr().(*net.TCPAddr).Port)
}

func TestListenAndServeStats(t *testing.T) {
	testIns := "testInstance"
	ctx, cancelFunc := context.WithCancel(context.Background())
	s, dataChan := testServer(ctx, t, testIns, &fakeLameduckLister{})
	defer cancelFunc()

	urlsAndExpectedResponse := map[string]string{
		"/":            OK,
		"/instance":    "testInstance",
		"/lameduck":    "false",
		"/healthcheck": OK,
	}
	for url, expectedResponse := range urlsAndExpectedResponse {
		if response, _ := get(t, s.ln, url); response != expectedResponse {
			t.Errorf("Didn't get the expected response for URL '%s'. Got: %s, Expected: %s", url, response, expectedResponse)
		}
	}
	// Sleep for the export interval and a second extra to allow for the stats to
	// come in.
	time.Sleep(s.statsInterval)
	time.Sleep(time.Second)

	// Build a map of expected URL stats
	expectedURLStats := make(map[string]int64)
	for url := range urlsAndExpectedResponse {
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

func TestLameduckingTestInstance(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	s, _ := testServer(ctx, t, "testInstance", &fakeLameduckLister{})
	defer cancelFunc()

	if resp, _ := get(t, s.ln, "lameduck"); !strings.Contains(resp, "false") {
		t.Errorf("Didn't get the expected response for the URL '/lameduck'. got: %q, want it to contain: %q", resp, "false")
	}
	if resp, status := get(t, s.ln, "healthcheck"); resp != OK || status != "200 OK" {
		t.Errorf("Didn't get the expected response for the URL '/healthcheck'. got: %q, %q , want: %q, %q", resp, status, OK, "200 OK")
	}

	s.ldLister = &fakeLameduckLister{[]string{"testInstance"}, nil}

	if resp, _ := get(t, s.ln, "lameduck"); !strings.Contains(resp, "true") {
		t.Errorf("Didn't get the expected response for the URL '/lameduck'. got: %q, want it to contain: %q", resp, "true")
	}
	if _, status := get(t, s.ln, "healthcheck"); status != "503 Service Unavailable" {
		t.Errorf("Didn't get the expected response for the URL '/healthcheck'. got: %q , want: %q", status, "200 OK")
	}
}

func TestLameduckListerNil(t *testing.T) {
	expectedErrMsg := "not initialized"

	ctx, cancelFunc := context.WithCancel(context.Background())
	s, _ := testServer(ctx, t, "testInstance", nil)
	defer cancelFunc()

	if resp, status := get(t, s.ln, "lameduck"); !strings.Contains(resp, expectedErrMsg) || status != "200 OK" {
		t.Errorf("Didn't get the expected response for the URL '/lameduck'. got: %q, %q. want it to contain: %q, %q", resp, status, expectedErrMsg, "200 OK")
	}
	if resp, status := get(t, s.ln, "healthcheck"); resp != OK || status != "200 OK" {
		t.Errorf("Didn't get the expected response for the URL '/healthcheck'. got: %q, %q , want: %q, %q", resp, status, OK, "200 OK")
	}
}
