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

package metrics

import (
	"testing"
)

func TestIsString(t *testing.T) {
	for _, test := range []struct {
		desc  string
		input Value
		want  bool
	}{
		{
			desc:  "Test with float",
			input: NewFloat(45.0),
			want:  false,
		},
		{
			desc:  "Test with string",
			input: NewString("test-string"),
			want:  true,
		},
		{
			desc: "Test with nil",
			want: false,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			isString := IsString(test.input)
			if isString != test.want {
				t.Errorf("IsString(%v)=%v, want=%v", test.input, isString, test.want)
			}
		})
	}
}
