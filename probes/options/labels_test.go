// Copyright 2017-2020 Google Inc.
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

package options

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	configpb "github.com/google/cloudprober/probes/proto"
)

var configWithAdditionalLabels = &configpb.ProbeDef{
	AdditionalLabel: []*configpb.AdditionalLabel{
		{
			Key:   proto.String("src_zone"),
			Value: proto.String("zoneA"),
		},
		{
			Key:   proto.String("dst_zone"),
			Value: proto.String("@target.label.zone@"),
		},
		{
			Key:   proto.String("dst_name"),
			Value: proto.String("@target.name@"),
		},
	},
}

func TestParseAdditionalLabel(t *testing.T) {
	expectedAdditionalLabels := []*AdditionalLabel{
		{
			Key:   "src_zone",
			Value: "zoneA",
		},
		{
			Key:             "dst_zone",
			TargetLabelType: TargetLabel,
			TargetLabelKey:  "zone",
			LabelForTarget:  make(map[string]string),
		},
		{
			Key:             "dst_name",
			TargetLabelType: TargetName,
			LabelForTarget:  make(map[string]string),
		},
	}

	aLabels := parseAdditionalLabels(configWithAdditionalLabels)

	// Verify that we got the correct additional lables and also update them while
	// iterating over them.
	for i, al := range aLabels {
		if !reflect.DeepEqual(al, expectedAdditionalLabels[i]) {
			t.Errorf("Additional labels not parsed correctly. Got=%v, Wanted=%v", al, expectedAdditionalLabels[i])
		}
	}
}

func TestUpdateAdditionalLabel(t *testing.T) {
	aLabels := parseAdditionalLabels(configWithAdditionalLabels)

	// Verify that we got the correct additional lables and also update them while
	// iterating over them.
	for _, al := range aLabels {
		al.UpdateForTarget("target1", map[string]string{})
		al.UpdateForTarget("target2", map[string]string{"zone": "zoneB"})
	}

	expectedLabels := map[string][][2]string{
		"target1": {{"src_zone", "zoneA"}, {"dst_zone", ""}, {"dst_name", "target1"}},
		"target2": {{"src_zone", "zoneA"}, {"dst_zone", "zoneB"}, {"dst_name", "target2"}},
	}

	for target, labels := range expectedLabels {
		var gotLabels [][2]string

		for _, al := range aLabels {
			k, v := al.KeyValueForTarget(target)
			gotLabels = append(gotLabels, [2]string{k, v})
		}

		if !reflect.DeepEqual(gotLabels, labels) {
			t.Errorf("Didn't get expected labels for the target: %s. Got=%v, Expected=%v", target, gotLabels, labels)
		}
	}
}
