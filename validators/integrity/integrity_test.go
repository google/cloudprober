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

package integrity

import (
	"strings"
	"testing"

	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/validators/integrity/proto"
)

func TestInvalidConfig(t *testing.T) {
	testConfig := &configpb.Validator{}
	v := Validator{}
	err := v.Init(testConfig, &logger.Logger{})
	if err == nil {
		t.Errorf("v.Init(%v, l): expected error but got nil", testConfig)
	}
}

func verifyValidate(t *testing.T, v Validator, testPattern string) {
	t.Helper()

	rows := []struct {
		respBody []byte
		expected bool
		wantErr  bool
	}{
		{
			// Response smaller than the pattern but bytes matche the pattern bytes.
			// It should pass when the pattern is specified (TestPatternString)
			// and fail when pattern is supposed to be derived from the payload by
			// looking at first N bytes (TestPatternNumBytes).
			respBody: append([]byte{}, []byte(testPattern[:3])...),
			expected: v.patternNumBytes == 0, // Pass when pattern num bytes are not given.
			wantErr:  v.patternNumBytes > 3,
		},
		{
			respBody: []byte(strings.Repeat(testPattern, 4)), // "test-ctest-ctest-ctest-c"
			expected: true,
		},
		{
			respBody: []byte(strings.Repeat(testPattern, 4) + "-123"), // "test-ctest-ctest-ctest-c-123"
			expected: false,
		},
		{
			// "test-ctest-c" with partial testPattern in the end.
			respBody: append([]byte(strings.Repeat(testPattern, 2)), testPattern[:2]...),
			expected: true,
		},
	}

	for i, r := range rows {
		result, err := v.Validate(nil, r.respBody)
		if (err != nil) != r.wantErr {
			t.Errorf("v.Validate(nil, %s), row #%d: err=%v, expectedError(bool)=%v", string(r.respBody), i, err, r.wantErr)
		}
		if result != r.expected {
			t.Errorf("v.Validate(nil, %s), row #%d: result=%v expected=%v", string(r.respBody), i, result, r.expected)
		}
	}
}

func TestPatternString(t *testing.T) {
	testPattern := "test-c"

	// Test initializing with pattern string.
	testConfig := &configpb.Validator{
		Pattern: &configpb.Validator_PatternString{
			PatternString: testPattern,
		},
	}

	v := Validator{}
	err := v.Init(testConfig, &logger.Logger{})
	if err != nil {
		t.Errorf("v.Init(%v, l): got error: %v", testConfig, err)
	}

	if string(v.pattern) != testPattern {
		t.Errorf("v.Init(%v): v.patternString=%s, expected=%s", testConfig, string(v.pattern), testPattern)
	}

	verifyValidate(t, v, testPattern)
}

func TestPatternNumBytes(t *testing.T) {
	testNumBytes := int32(8)

	// Test initializing with pattern with prefix num bytes.
	v, err := PatternNumBytesValidator(testNumBytes, &logger.Logger{})
	if err != nil {
		t.Errorf("PatternNumBytesValidator(%d, l): got error: %v", testNumBytes, err)
	}

	if v.patternNumBytes != testNumBytes {
		t.Errorf("PatternNumBytesValidator(%d, l): v.patternNumBytes=%d, expected=%d", testNumBytes, v.patternNumBytes, testNumBytes)
	}

	// 8-byte long test pattern to be used for respBody generation
	testPattern := "njk1120sasnl123"[:8]
	verifyValidate(t, *v, testPattern)
}
