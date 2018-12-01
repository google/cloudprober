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

// Package regex provides regex validator for the Cloudprober's
// validator framework.
package regex

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/google/cloudprober/logger"
)

// Validator implements a regex validator.
type Validator struct {
	r *regexp.Regexp
	l *logger.Logger
}

// Init initializes the regex validator.
// It compiles the regex in the configuration and returns an error if regex
// doesn't compile for some reason.
func (v *Validator) Init(config interface{}, l *logger.Logger) error {
	regexStr, ok := config.(string)
	if !ok {
		return fmt.Errorf("%v is not a valid regex validator config", config)
	}
	if regexStr == "" {
		return errors.New("validator regex string cannot be empty")
	}

	r, err := regexp.Compile(regexStr)
	if err != nil {
		return fmt.Errorf("error compiling the given regex (%s): %v", regexStr, err)
	}

	v.r = r
	v.l = l
	return nil
}

// Validate the provided responseBody and return true if responseBody matches
// the configured regex.
func (v *Validator) Validate(unusedResponseObj interface{}, responseBody []byte) (bool, error) {
	return v.r.Match(responseBody), nil
}
