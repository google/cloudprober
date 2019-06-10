// Copyright 2018-2019 Google Inc.
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

// Validator interface represents a validator.
//
// A validator runs a test on the provided input, usually the probe response,
// and returns the test result. If test cannot be run successfully for some
// reason (e.g. for malformed input), an error is returned.
type Validator interface {
	Init(config interface{}, l *logger.Logger) error
	Validate(responseObject interface{}, responseBody []byte) (bool, error)
}

// ValidatorWithName encapsulate a Validator and its name.
type ValidatorWithName struct {
	Name      string
	Validator Validator
}

// Validate is simply a wrapper around the encapsulated Validator's Validate
// method.
func (vwn *ValidatorWithName) Validate(responseObject interface{}, responseBody []byte) (bool, error) {
	return vwn.Validator.Validate(responseObject, responseBody)
}

// Init initializes the validators defined in the config.
func Init(validatorConfs []*configpb.Validator, l *logger.Logger) ([]*ValidatorWithName, error) {
	var validators []*ValidatorWithName
	names := make(map[string]bool)

	for _, vc := range validatorConfs {
		if names[vc.GetName()] {
			return nil, fmt.Errorf("validator %s is defined twice", vc.GetName())
		}

		v, err := initValidator(vc, l)
		if err != nil {
			return nil, err
		}

		validators = append(validators, &ValidatorWithName{vc.GetName(), v})
		names[vc.GetName()] = true
	}

	return validators, nil
}

func initValidator(validatorConf *configpb.Validator, l *logger.Logger) (validator Validator, err error) {
	var c interface{}

	switch validatorConf.Type.(type) {
	case *configpb.Validator_HttpValidator:
		validator = &http.Validator{}
		c = validatorConf.GetHttpValidator()
	case *configpb.Validator_IntegrityValidator:
		validator = &integrity.Validator{}
		c = validatorConf.GetIntegrityValidator()
	case *configpb.Validator_Regex:
		validator = &regex.Validator{}
		c = validatorConf.GetRegex()
	default:
		err = fmt.Errorf("unknown validator type: %v", validatorConf.Type)
		return
	}

	err = validator.Init(c, l)
	return
}

// RunValidators runs the list of validators on the given response and
// responseBody, updates the given validationFailure map and returns the list
// of failures.
func RunValidators(vs []*ValidatorWithName, resp interface{}, respBody []byte, validationFailure *metrics.Map, l *logger.Logger) []string {
	var failures []string

	for _, v := range vs {
		success, err := v.Validate(resp, respBody)
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
func ValidationFailureMap(vs []*ValidatorWithName) *metrics.Map {
	m := metrics.NewMap("validator", metrics.NewInt(0))
	// Initialize validation failure map with validator keys, so that we always
	// export the metrics.
	for _, v := range vs {
		m.IncKeyBy(v.Name, metrics.NewInt(0))
	}
	return m
}
