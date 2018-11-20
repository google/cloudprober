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

// Package integrity provides data integrity validator for the Cloudprober's
// validator framework.
package integrity

import (
	"fmt"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/probes/probeutils"
	configpb "github.com/google/cloudprober/validators/integrity/proto"
)

// Validator implements an integrity validator.
type Validator struct {
	patternString   string
	patternNumBytes int32

	l *logger.Logger
}

// Init initializes the integrity validator.
func (v *Validator) Init(config interface{}, l *logger.Logger) error {
	c, ok := config.(*configpb.Validator)
	if !ok {
		return fmt.Errorf("%v is not a valid integrity validator config", config)
	}

	switch c.GetPattern().(type) {
	case *configpb.Validator_PatternString:
		v.patternString = c.GetPatternString()
	case *configpb.Validator_PatternNumBytes:
		v.patternNumBytes = c.GetPatternNumBytes()
	}

	if v.patternString == "" && v.patternNumBytes == 0 {
		return fmt.Errorf("bad integrity validator config (%v): one of pattern_string and pattern_num_bytes should be set", c)
	}

	v.l = l
	return nil
}

// Validate validates the provided responseBody for data integrity errors, for
// example, data corruption.
func (v *Validator) Validate(unusedResponseObj interface{}, responseBody []byte) (bool, error) {
	pattern := []byte(v.patternString)
	if len(pattern) == 0 {
		if len(responseBody) < int(v.patternNumBytes) {
			return false, fmt.Errorf("response (%s) is smaller than the number of pattern bytes (%d)", responseBody, v.patternNumBytes)
		}
		pattern = responseBody[:v.patternNumBytes]
	}

	err := probeutils.VerifyPayloadPattern(responseBody, pattern)

	if err != nil {
		v.l.Error(err.Error())
		return false, nil
	}

	return true, nil
}
