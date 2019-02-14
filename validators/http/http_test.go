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

package http

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/validators/http/proto"
)

func TestParseStatusCodeConfig(t *testing.T) {
	testStr := "302,200-299,403"
	numRanges, err := parseStatusCodeConfig(testStr)

	if err != nil {
		t.Errorf("parseStatusCodeConfig(%s): got error: %v", testStr, err)
	}

	expectedNR := []*numRange{
		&numRange{
			lower: 302,
			upper: 302,
		},
		&numRange{
			lower: 200,
			upper: 299,
		},
		&numRange{
			lower: 403,
			upper: 403,
		},
	}

	if len(numRanges) != len(expectedNR) {
		t.Errorf("parseStatusCodeConfig(%s): len(numRanges): %d, expected: %d", testStr, len(numRanges), len(expectedNR))
	}

	for i, nr := range numRanges {
		if !reflect.DeepEqual(nr, expectedNR[i]) {
			t.Errorf("parseStatusCodeConfig(%s): nr[%d]: %v, expected[%d]: %v", testStr, i, nr, i, expectedNR[i])
		}
	}

	// Verify that parsing invalid status code strings result in an error.
	invalidTestStr := []string{
		"30a,404",
		"301,299-200",
		"301,200-299-400",
	}
	for _, s := range invalidTestStr {
		numRanges, err := parseStatusCodeConfig(s)
		if err == nil {
			t.Errorf("parseStatusCodeConfig(%s): expected error but got response: %v", s, numRanges)
		}
	}
}

func TestLookupStatusCode(t *testing.T) {
	testStr := "302,200-299,403"
	numRanges, _ := parseStatusCodeConfig(testStr)

	var found bool
	for _, code := range []int{200, 204, 302, 403} {
		found = lookupStatusCode(code, numRanges)
		if !found {
			t.Errorf("lookupStatusCode(%d, nr): %v, expected: true", code, found)
		}
	}

	for _, code := range []int{404, 500, 502, 301} {
		found = lookupStatusCode(code, numRanges)
		if found {
			t.Errorf("lookupStatusCode(%d, nr): %v, expected: false", code, found)
		}
	}
}

func TestInit(t *testing.T) {
	testConfig := &configpb.Validator{
		SuccessStatusCodes: proto.String("200-299,301,302,404"),
		FailureStatusCodes: proto.String("403,404,500-502"),
	}

	v := &Validator{}
	err := v.Init(testConfig, &logger.Logger{})
	if err != nil {
		t.Errorf("Init(%v, l): err: %v", testConfig, err)
	}

	for _, code := range []int{200, 204, 302} {
		expected := true
		res := &http.Response{
			StatusCode: code,
		}
		result, _ := v.Validate(res, nil)
		if result != expected {
			t.Errorf("v.Validate(&http.Response{StatusCode: %d}, nil): %v, expected: %v", code, result, expected)
		}
	}

	for _, code := range []int{501, 502, 403, 404} {
		expected := false
		res := &http.Response{
			StatusCode: code,
		}
		result, _ := v.Validate(res, nil)
		if result != expected {
			t.Errorf("v.Validate(&http.Response{StatusCode: %d}, nil): %v, expected: %v", code, result, expected)
		}
	}
}
