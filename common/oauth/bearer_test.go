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

package oauth

import (
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	configpb "github.com/google/cloudprober/common/oauth/proto"
)

var global struct {
	callCounter int
	mu          sync.RWMutex
}

func incCallCounter() {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.callCounter++
}

func callCounter() int {
	global.mu.RLock()
	defer global.mu.RUnlock()
	return global.callCounter
}

func testTokenFromFile(c *configpb.BearerToken) (string, error) {
	incCallCounter()
	return c.GetFile() + "_file_token", nil
}

func testTokenFromCmd(c *configpb.BearerToken) (string, error) {
	incCallCounter()
	return c.GetCmd() + "_cmd_token", nil
}

func testTokenFromGCEMetadata(c *configpb.BearerToken) (string, error) {
	incCallCounter()
	return c.GetGceServiceAccount() + "_gce_token", nil
}

func TestNewBearerToken(t *testing.T) {
	getTokenFromFile = testTokenFromFile
	getTokenFromCmd = testTokenFromCmd
	getTokenFromGCEMetadata = testTokenFromGCEMetadata

	testConfig := "file: \"f\""
	verifyBearerTokenSource(t, true, testConfig, "f_file_token")

	// Disable caching by setting refresh_interval_sec to 0.
	testConfig = "file: \"f\"\nrefresh_interval_sec: 0"
	verifyBearerTokenSource(t, false, testConfig, "f_file_token")

	testConfig = "cmd: \"c\""
	verifyBearerTokenSource(t, true, testConfig, "c_cmd_token")

	testConfig = "gce_service_account: \"default\""
	verifyBearerTokenSource(t, true, testConfig, "default_gce_token")
}

func verifyBearerTokenSource(t *testing.T, cacheEnabled bool, testConfig, expectedToken string) {
	t.Helper()

	testC := &configpb.BearerToken{}
	err := proto.UnmarshalText(testConfig, testC)
	if err != nil {
		t.Fatalf("error parsing test config (%s): %v", testConfig, err)
	}

	// Call counter should always increase during token source creation.
	expectedC := callCounter() + 1

	cts, err := newBearerTokenSource(testC, nil)
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
	}

	cc := callCounter()
	if cc != expectedC {
		t.Errorf("unexpected call counter: got=%d, expected=%d", cc, expectedC)
	}

	tok, err := cts.Token()
	if err != nil {
		t.Errorf("unexpected error while retrieving token from config (%s): %v", testConfig, err)
	}

	if tok.AccessToken != expectedToken {
		t.Errorf("Got token: %s, expected: %s", tok.AccessToken, expectedToken)
	}

	// Call counter will increase after Token call only if caching is disabled.
	if !cacheEnabled {
		expectedC++
	}
	cc = callCounter()

	if cc != expectedC {
		t.Errorf("unexpected call counter: got=%d, expected=%d", cc, expectedC)
	}
}

var (
	calledTestTokenOnce   bool
	calledTestTokenOnceMu sync.Mutex
)

func testTokenRefresh(c *configpb.BearerToken) (string, error) {
	calledTestTokenOnceMu.Lock()
	defer calledTestTokenOnceMu.Unlock()
	if calledTestTokenOnce {
		return "new-token", nil
	}
	calledTestTokenOnce = true
	return "old-token", nil
}

// TestRefreshCycle verifies that token gets refreshed after the refresh
// cycle.
func TestRefreshCycle(t *testing.T) {
	getTokenFromCmd = testTokenRefresh
	// Disable caching by setting refresh_interval_sec to 0.
	testConfig := "cmd: \"c\"\nrefresh_interval_sec: 1"

	testC := &configpb.BearerToken{}
	err := proto.UnmarshalText(testConfig, testC)
	if err != nil {
		t.Fatalf("error parsing test config (%s): %v", testConfig, err)
	}

	ts, err := newBearerTokenSource(testC, nil)
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
	}

	tok, err := ts.Token()
	if err != nil {
		t.Errorf("unexpected error while retrieving token from config (%s): %v", testConfig, err)
	}

	oldToken := "old-token"
	newToken := "new-token"

	if tok.AccessToken != oldToken {
		t.Errorf("ts.Token(): got=%s, expected=%s", tok, oldToken)
	}

	time.Sleep(5 * time.Second)

	tok, err = ts.Token()
	if err != nil {
		t.Errorf("unexpected error while retrieving token from config (%s): %v", testConfig, err)
	}

	if tok.AccessToken != newToken {
		t.Errorf("ts.Token(): got=%s, expected=%s", tok, newToken)
	}
}
