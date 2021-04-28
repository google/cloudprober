// Copyright 2018-2019 The Cloudprober Authors.
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

// Package validators provides an entrypoint for the cloudprober's validators
// framework.
package validators

import (
	"fmt"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/validators/http"
	"github.com/google/cloudprober/validators/integrity"
	configpb "github.com/google/cloudprober/validators/proto"
	"github.com/google/cloudprober/validators/regex"
)

// Validator implements a validator.
//
// A validator runs a test on the provided input, usually the probe response,
// and returns the test result. If test cannot be run successfully for some
// reason (e.g. for malformed input), an error is returned.
type Validator struct {
	Name     string
	Validate func(input *Input) (bool, error)
}

// Init initializes the validators defined in the config.
func Init(validatorConfs []*configpb.Validator, l *logger.Logger) ([]*Validator, error) {
	var validators []*Validator
	names := make(map[string]bool)

	for _, vc := range validatorConfs {
		if names[vc.GetName()] {
			return nil, fmt.Errorf("validator %s is defined twice", vc.GetName())
		}

		v, err := initValidator(vc, l)
		if err != nil {
			return nil, err
		}

		validators = append(validators, v)
		names[vc.GetName()] = true
	}

	return validators, nil
}

func initValidator(validatorConf *configpb.Validator, l *logger.Logger) (validator *Validator, err error) {
	validator = &Validator{Name: validatorConf.GetName()}

	switch validatorConf.Type.(type) {
	case *configpb.Validator_HttpValidator:
		v := &http.Validator{}
		if err := v.Init(validatorConf.GetHttpValidator(), l); err != nil {
			return nil, err
		}
		validator.Validate = func(input *Input) (bool, error) {
			return v.Validate(input.Response, input.ResponseBody)
		}
		return

	case *configpb.Validator_IntegrityValidator:
		v := &integrity.Validator{}
		if err := v.Init(validatorConf.GetIntegrityValidator(), l); err != nil {
			return nil, err
		}
		validator.Validate = func(input *Input) (bool, error) {
			return v.Validate(input.ResponseBody)
		}
		return

	case *configpb.Validator_Regex:
		v := &regex.Validator{}
		if err := v.Init(validatorConf.GetRegex(), l); err != nil {
			return nil, err
		}
		validator.Validate = func(input *Input) (bool, error) {
			return v.Validate(input.ResponseBody)
		}
		return
	default:
		err = fmt.Errorf("unknown validator type: %v", validatorConf.Type)
		return
	}
}

// Input encapsulates the input for validators.
type Input struct {
	Response     interface{}
	ResponseBody []byte
}

// RunValidators runs the list of validators on the given response and
// responseBody, updates the given validationFailure map and returns the list
// of failures.
func RunValidators(vs []*Validator, input *Input, validationFailure *metrics.Map, l *logger.Logger) []string {
	var failures []string

	for _, v := range vs {
		success, err := v.Validate(input)
		if err != nil {
			l.Error("Error while running the validator ", v.Name, ": ", err.Error())
			continue
		}
		if !success {
			validationFailure.IncKey(v.Name)
			failures = append(failures, v.Name)
		}
	}

	return failures
}

// ValidationFailureMap returns an initialized validation failures map.
func ValidationFailureMap(vs []*Validator) *metrics.Map {
	m := metrics.NewMap("validator", metrics.NewInt(0))
	// Initialize validation failure map with validator keys, so that we always
	// export the metrics.
	for _, v := range vs {
		m.IncKeyBy(v.Name, metrics.NewInt(0))
	}
	return m
}
