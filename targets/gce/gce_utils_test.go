// Copyright 2017-2019 Google Inc.
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

package gce

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	rdspb "github.com/google/cloudprober/targets/rds/proto"
)

func TestParseLabels(t *testing.T) {
	tests := []struct {
		desc       string
		label      string
		shouldFail bool
		want       []*rdspb.Filter
	}{
		{
			"Valid label should succeed",
			"k:v",
			false,
			[]*rdspb.Filter{
				{
					Key:   proto.String("labels.k"),
					Value: proto.String("v"),
				},
			},
		},
		{
			"Multiple separators should fail",
			"k:v:t",
			true,
			nil,
		},
		{
			"No separator should fail",
			"kv",
			true,
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got, err := parseLabels([]string{test.label})
			if test.shouldFail && err == nil {
				t.Errorf("parseLabels() error got:nil, want:error")
			} else if !test.shouldFail && err != nil {
				t.Errorf("parseLabels()  error got:%s, want:nil", err)
			}
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("parseLabels()  got:%s, want:%s", got, test.want)
			}
		})
	}
}
