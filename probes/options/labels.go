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
	"regexp"
	"strings"

	configpb "github.com/google/cloudprober/probes/proto"
)

// TargetLabelType for target based additional labels
type TargetLabelType int

// TargetLabelType enum values.
const (
	NotTargetLabel TargetLabelType = iota
	TargetLabel
	TargetName
)

var targetLabelRegex = regexp.MustCompile(`@target.label.(.*)@`)

// AdditionalLabel encapsulates additional labels to attach to probe results.
type AdditionalLabel struct {
	Key string

	Value           string // static value
	TargetLabelKey  string // from target
	TargetLabelType TargetLabelType

	// This map will allow for quick label lookup for a target. It will be
	// updated by the probe while updating targets.
	LabelForTarget map[string]string
}

// UpdateForTarget updates target-based label's value.
func (al *AdditionalLabel) UpdateForTarget(tname string, tLabels map[string]string) {
	if al.TargetLabelType == NotTargetLabel {
		return
	}

	if al.LabelForTarget == nil {
		al.LabelForTarget = make(map[string]string)
	}

	switch al.TargetLabelType {
	case TargetLabel:
		al.LabelForTarget[tname] = tLabels[al.TargetLabelKey]
	case TargetName:
		al.LabelForTarget[tname] = tname
	}
}

// KeyValueForTarget returns key, value pair for the given target.
func (al *AdditionalLabel) KeyValueForTarget(targetName string) (key, val string) {
	if al.Value != "" {
		return al.Key, al.Value
	}
	return al.Key, al.LabelForTarget[targetName]
}

func parseAdditionalLabels(p *configpb.ProbeDef) []*AdditionalLabel {
	var aLabels []*AdditionalLabel

	for _, pb := range p.GetAdditionalLabel() {
		al := &AdditionalLabel{
			Key: pb.GetKey(),
		}
		aLabels = append(aLabels, al)

		val := pb.GetValue()
		if !strings.Contains(val, "@") {
			al.Value = val
			continue
		}

		if val == "@target.name@" {
			al.TargetLabelType = TargetName
			al.LabelForTarget = make(map[string]string)
			continue
		}

		matches := targetLabelRegex.FindStringSubmatch(val)
		if len(matches) == 2 {
			al.TargetLabelType = TargetLabel
			al.TargetLabelKey = matches[1]
			al.LabelForTarget = make(map[string]string)
			continue
		}

		// If a match is not found and target contains an "@" character, use it
		// as it is.
		al.Value = val
	}

	return aLabels
}
