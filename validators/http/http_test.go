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
	"fmt"
	"net/http"
	"reflect"
	"regexp"
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

func TestLookupHTTPHeader(t *testing.T) {
	var header string

	headers := http.Header{"X-Success": []string{"some", "truly", "last"}}

	header = "X-Failure"
	if lookupHTTPHeader(headers, header, nil) != false {
		t.Errorf("lookupHTTPHeader(&%T%+v, %v, %v): true, expected false", headers, headers, header, nil)
	}

	header = "X-Success"
	if lookupHTTPHeader(headers, header, nil) != true {
		t.Errorf("lookupHTTPHeader(&%T%+v, %v, %v): false expected: true", headers, headers, header, nil)
	}

	r := regexp.MustCompile("badl[ya]")
	if lookupHTTPHeader(headers, header, r) != false {
		t.Errorf("lookupHTTPHeader(&%T%+v, %v, %v): true expected: false", headers, headers, header, r)
	}

	r = regexp.MustCompile("tr[ul]ly")
	if lookupHTTPHeader(headers, header, r) != true {
		t.Errorf("lookupHTTPHeader(&%T%+v, %v, %v): false expected: true", headers, headers, header, r)
	}

}

func TestValidateStatusCode(t *testing.T) {
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

func TestValidateHeaders(t *testing.T) {
	testConfig := &configpb.Validator{}

	v := &Validator{}
	err := v.Init(testConfig, &logger.Logger{})
	if err != nil {
		t.Errorf("Init(%v, l): err: %v", testConfig, err)
	}

	respStatus := http.StatusOK
	respHeader := http.Header{
		"X-Success": []string{"some", "truly", "last"},
		"X-Bad":     []string{"some-bad"},
	}

	for _, test := range []struct {
		sStatus       string
		fStatus       string
		sHeader       []string
		fHeader       []string
		wantInitError bool
		wantValid     bool
	}{
		{
			sHeader:       []string{"", ""},
			wantInitError: true, // No header name
		},
		{
			sHeader:       []string{"X-Success", "[bad_regex"},
			wantInitError: true, // Bad regex.
		},
		{
			sHeader:   []string{"X-Success", ""},
			wantValid: true,
		},
		{
			sHeader:   []string{"X-Success", "s[om]me"},
			wantValid: true,
		},
		{
			sHeader:   []string{"X-Success", "best"},
			wantValid: false,
		},
		{
			sHeader:   []string{"X-Success", ""},
			fHeader:   []string{"X-Bad", ""},
			wantValid: false, // Bad header is also present, so we fail.
		},
		{
			sHeader:   []string{"X-Success", ""},
			fHeader:   []string{"X-Bad", "really-bad"},
			wantValid: true, // Not bad enough.
		},
		{
			sHeader:   []string{"X-Success", ""},
			fHeader:   []string{"X-Bad", "some-bad"},
			wantValid: false, // Bad enough.
		},
		{
			sStatus:   "302",
			sHeader:   []string{"X-Success", ""},
			wantValid: false, // Fails because of status code mismatch: 200 vs 302.
		},
	} {
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {
			testConfig := &configpb.Validator{
				SuccessStatusCodes: proto.String(test.sStatus),
				FailureStatusCodes: proto.String(test.fStatus),
			}

			if test.sHeader != nil {
				testConfig.SuccessHeader = &configpb.Validator_Header{
					Name:       proto.String(test.sHeader[0]),
					ValueRegex: proto.String(test.sHeader[1]),
				}
			}

			if test.fHeader != nil {
				testConfig.FailureHeader = &configpb.Validator_Header{
					Name:       proto.String(test.fHeader[0]),
					ValueRegex: proto.String(test.fHeader[1]),
				}
			}

			v := &Validator{}
			err := v.Init(testConfig, &logger.Logger{})
			if (err != nil) != test.wantInitError {
				t.Errorf("Init(%v, l): got err: %v, wantError: %v", testConfig, err, test.wantInitError)
			}
			if err != nil {
				return
			}

			resp := &http.Response{
				Header:     respHeader,
				StatusCode: respStatus,
			}
			ok, err := v.Validate(resp, nil)

			if err != nil {
				t.Errorf("Error running validate (resp: %v): %v", resp, err)
			}

			if ok != test.wantValid {
				t.Errorf("validation passed: %v, wanted to pass: %v", ok, test.wantValid)
			}
		})
	}
}
