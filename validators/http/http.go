// Copyright 2018 The Cloudprober Authors.
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

// Package http provides an HTTP validator for the Cloudprober's validator
// framework.
package http

import (
	"errors"
	"fmt"
	nethttp "net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/validators/http/proto"
)

// Validator implements a validator for HTTP responses.
type Validator struct {
	c *configpb.Validator
	l *logger.Logger

	successStatusCodeRanges []*numRange
	failureStatusCodeRanges []*numRange
	successHeaderRegexp     *regexp.Regexp
	failureHeaderRegexp     *regexp.Regexp
}

type numRange struct {
	lower int
	upper int
}

func (nr *numRange) find(i int) bool {
	return i >= nr.lower && i <= nr.upper
}

// parseNumRange parses number range from the given string:
// for example:
//          200-299: &numRange{200, 299}
//          403:     &numRange{403, 403}
func parseNumRange(s string) (*numRange, error) {
	fields := strings.Split(s, "-")
	if len(fields) < 1 || len(fields) > 2 {
		return nil, fmt.Errorf("number range %s is not in correct format (200 or 100-199)", s)
	}

	lower, err := strconv.ParseInt(fields[0], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("got error while parsing the range's lower bound (%s): %v", fields[0], err)
	}

	// If there is only one number, set upper = lower.
	if len(fields) == 1 {
		return &numRange{int(lower), int(lower)}, nil
	}

	upper, err := strconv.ParseInt(fields[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("got error while parsing the range's upper bound (%s): %v", fields[1], err)
	}

	if upper < lower {
		return nil, fmt.Errorf("upper bound cannot be smaller than the lower bound (%s)", s)
	}

	return &numRange{int(lower), int(upper)}, nil
}

// parseStatusCodeConfig parses the status code config. Status codes are
// defined as a comma-separated list of integer or integer ranges, for
// example: 302,200-299.
func parseStatusCodeConfig(s string) ([]*numRange, error) {
	var statusCodeRanges []*numRange

	for _, codeStr := range strings.Split(s, ",") {
		nr, err := parseNumRange(codeStr)
		if err != nil {
			return nil, err
		}
		statusCodeRanges = append(statusCodeRanges, nr)
	}
	return statusCodeRanges, nil
}

// lookupStatusCode looks up a given status code in status code map and status
// code ranges.
func lookupStatusCode(statusCode int, statusCodeRanges []*numRange) bool {
	// Look for the statusCode in statusCodeRanges.
	for _, cr := range statusCodeRanges {
		if cr.find(statusCode) {
			return true
		}
	}

	return false
}

// lookupHTTPHeader looks up for the given header in the HTTP response. It
// returns true on the first match. If valueRegex is omitted - check for header
// existence only.
func lookupHTTPHeader(headers nethttp.Header, expectedHeader string, valueRegexp *regexp.Regexp) bool {
	values, found := headers[expectedHeader]
	if !found {
		return false
	}

	// Return true if not interested in header's value.
	if valueRegexp == nil {
		return true
	}

	for _, value := range values {
		if valueRegexp.MatchString(value) {
			return true
		}
	}

	return false
}

func (v *Validator) initHeaderValidators(c *configpb.Validator) error {
	parseHeader := func(h *configpb.Validator_Header) (*regexp.Regexp, error) {
		if h == nil {
			return nil, nil
		}
		if h.GetName() == "" {
			return nil, errors.New("header name cannot be empty")
		}
		if h.GetValueRegex() == "" {
			return nil, nil
		}
		return regexp.Compile(h.GetValueRegex())
	}

	var err error

	if v.successHeaderRegexp, err = parseHeader(c.GetSuccessHeader()); err != nil {
		return fmt.Errorf("invalid-success-header: %v", err)
	}

	if v.failureHeaderRegexp, err = parseHeader(c.GetFailureHeader()); err != nil {
		return fmt.Errorf("invalid-failure-header: %v", err)
	}

	return nil
}

// Init initializes the HTTP validator.
func (v *Validator) Init(config interface{}, l *logger.Logger) error {
	c, ok := config.(*configpb.Validator)
	if !ok {
		return fmt.Errorf("%v is not a valid HTTP validator config", config)
	}

	v.c = c
	v.l = l

	var err error
	if c.GetSuccessStatusCodes() != "" {
		v.successStatusCodeRanges, err = parseStatusCodeConfig(c.GetSuccessStatusCodes())
		if err != nil {
			return err
		}
	}

	if c.GetFailureStatusCodes() != "" {
		v.failureStatusCodeRanges, err = parseStatusCodeConfig(c.GetFailureStatusCodes())
		if err != nil {
			return err
		}
	}

	return v.initHeaderValidators(c)
}

// Validate the provided input and return true if input is valid. Validate
// expects the input to be of the type: *http.Response. Note that it doesn't
// use the string input, it's part of the function signature to satisfy
// Validator interface.
func (v *Validator) Validate(input interface{}, unused []byte) (bool, error) {
	res, ok := input.(*nethttp.Response)
	if !ok {
		return false, fmt.Errorf("input %v is not of type http.Response", input)
	}

	if v.c.GetFailureStatusCodes() != "" {
		if lookupStatusCode(res.StatusCode, v.failureStatusCodeRanges) {
			return false, nil
		}
	}

	if failureHeader := v.c.GetFailureHeader(); failureHeader != nil {
		if lookupHTTPHeader(res.Header, failureHeader.GetName(), v.failureHeaderRegexp) {
			return false, nil
		}
	}

	if v.c.GetSuccessStatusCodes() != "" {
		if !lookupStatusCode(res.StatusCode, v.successStatusCodeRanges) {
			return false, nil
		}
	}

	if successHeader := v.c.GetSuccessHeader(); successHeader != nil {
		if !lookupHTTPHeader(res.Header, successHeader.GetName(), v.successHeaderRegexp) {
			return false, nil
		}
	}

	return true, nil
}
