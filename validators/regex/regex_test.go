// Copyright 2018 Google Inc.
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

package regex

import (
	"testing"

	"github.com/google/cloudprober/logger"
)

func TestInvalidConfig(t *testing.T) {
	// Empty config
	testConfig := ""
	v := Validator{}
	err := v.Init(testConfig, &logger.Logger{})
	if err == nil {
		t.Errorf("v.Init(%s, l): expected error but got nil", testConfig)
	}

	// Invalid regex as Go regex doesn't support negative lookaheads.
	testConfig = "(?!cloudprober)"
	v = Validator{}
	err = v.Init(testConfig, &logger.Logger{})
	if err == nil {
		t.Errorf("v.Init(%s, l): expected error but got nil", testConfig)
	}
}

func verifyValidate(t *testing.T, respBody []byte, regexStr string, expected bool) {
	t.Helper()
	// Test initializing with pattern string.
	v := Validator{}
	err := v.Init(regexStr, &logger.Logger{})
	if err != nil {
		t.Errorf("v.Init(%s, l): got error: %v", regexStr, err)
	}

	result, err := v.Validate(nil, respBody)
	if err != nil {
		t.Errorf("v.Validate(nil, %s): got error: %v", string(respBody), err)
	}

	if result != expected {
		if err != nil {
			t.Errorf("v.Validate(nil, %s): result: %v, expected: %v", string(respBody), result, expected)
		}
	}
}

func TestPatternString(t *testing.T) {
	rows := []struct {
		regex    string
		respBody []byte
		expected bool
	}{
		{
			regex:    "cloud.*",
			respBody: []byte("cloudprober"),
			expected: true,
		},
		{
			regex:    "[Cc]loud.*",
			respBody: []byte("Cloudprober"),
			expected: false,
		},
	}

	for _, r := range rows {
		verifyValidate(t, r.respBody, r.regex, r.expected)
	}

}
