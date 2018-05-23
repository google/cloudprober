// Copyright 2017 Google Inc.
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

package surfacers

import (
	"testing"

	surfacerpb "github.com/google/cloudprober/surfacers/proto"
)

func TestDefaultConfig(t *testing.T) {
	s, err := Init([]*surfacerpb.SurfacerDef{})
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != len(defaultSurfacers) {
		t.Errorf("Didn't get default surfacers for no config")
	}
}

func TestEmptyConfig(t *testing.T) {
	s, err := Init([]*surfacerpb.SurfacerDef{&surfacerpb.SurfacerDef{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 0 {
		t.Errorf("Got surfacers for zero config: %v", s)
	}
}
