// Copyright 2019 Google Inc.
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

package validators

import (
	"reflect"
	"testing"

	"github.com/google/cloudprober/logger"
)

type testValidator struct {
	Succeed bool
}

func (tv *testValidator) Init(config interface{}, l *logger.Logger) error {
	return nil
}

func (tv *testValidator) Validate(responseObject interface{}, responseBody []byte) (bool, error) {
	if tv.Succeed {
		return true, nil
	}
	return false, nil
}

var testValidators = []*ValidatorWithName{
	{
		Name:      "test-v1",
		Validator: &testValidator{true},
	},
	{
		Name:      "test-v2",
		Validator: &testValidator{false},
	},
}

func TestRunValidators(t *testing.T) {
	vfMap := ValidationFailureMap(testValidators)
	failures := RunValidators(testValidators, nil, nil, vfMap, nil)

	if vfMap.GetKey("test-v1").Int64() != 0 {
		t.Error("Got unexpected test-v1 validation failure.")
	}

	if vfMap.GetKey("test-v2").Int64() != 1 {
		t.Errorf("Didn't get expected test-v2 validation failure.")
	}

	if len(failures) != 1 && failures[0] != "test-v2" {
		t.Errorf("Didn't get expected validation failures. Expected: {\"test-v2\"}, Got: %v", failures)
	}
}

func TestValidatorFailureMap(t *testing.T) {
	vfMap := ValidationFailureMap(testValidators)

	expectedKeys := []string{"test-v1", "test-v3"}
	if reflect.DeepEqual(vfMap.Keys(), expectedKeys) {
		t.Errorf("Didn't get expected keys in the mao. Got: %s, Expected: %v", vfMap.Keys(), expectedKeys)
	}
}
