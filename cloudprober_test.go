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

package cloudprober

import (
	"strings"
	"testing"
)

func TestParsePort(t *testing.T) {
	// test if it parses just a number in string format
	port, _ := parsePort("1234")
	expectedPort := int64(1234)
	if port != expectedPort {
		t.Errorf("parsePort(\"%d\") = %d; want %d", expectedPort, port, expectedPort)
	}

	// test if it parses full URL
	testStr := "tcp://10.1.1.4:9313"
	port, _ = parsePort(testStr)
	expectedPort = int64(9313)
	if port != expectedPort {
		t.Errorf("parsePort(\"%s\") = %d; want %d", testStr, port, expectedPort)
	}

	// test if it detects absent port in URL
	testStr = "tcp://10.1.1.4"
	_, err := parsePort(testStr)
	errStr := "no port specified in URL"
	if err != nil && !strings.Contains(err.Error(), errStr) {
		t.Errorf("parsePort(\"%s\") doesn't return \"%s\" error, however found error: %s", testStr, "no port specified in URL", err.Error())
	} else if err == nil {
		t.Errorf("parsePort(\"%s\") should return \"%s\" error, however no errors found", testStr, "no port specified in URL")
	}
}
