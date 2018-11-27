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

// Package validators provides an entrypoint for the cloudprober's validators
// framework.
package validators

import (
	"fmt"

	"github.com/google/cloudprober/logger"
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

// Init initializes the validators defined in the config.
func Init(validatorConfs []*configpb.Validator, l *logger.Logger) (map[string]Validator, error) {
	validators := make(map[string]Validator)

	for _, vc := range validatorConfs {
		v, err := initValidator(vc, l)
		if err != nil {
			return nil, err
		}
		validators[vc.GetName()] = v
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
	case *configpb.Validator_RegexValidator:
		validator = &regex.Validator{}
		c = validatorConf.GetRegexValidator()
	default:
		err = fmt.Errorf("unknown validator type: %v", validatorConf.Type)
		return
	}

	err = validator.Init(c, l)
	return
}
