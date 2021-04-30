// Copyright 2017 The Cloudprober Authors.
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
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/surfacers/file"
	fileconfigpb "github.com/google/cloudprober/surfacers/file/proto"
	surfacerpb "github.com/google/cloudprober/surfacers/proto"
)

func TestDefaultConfig(t *testing.T) {
	s, err := Init(context.Background(), []*surfacerpb.SurfacerDef{})
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != len(defaultSurfacers) {
		t.Errorf("Didn't get default surfacers for no config")
	}
}

func TestEmptyConfig(t *testing.T) {
	s, err := Init(context.Background(), []*surfacerpb.SurfacerDef{&surfacerpb.SurfacerDef{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 0 {
		t.Errorf("Got surfacers for zero config: %v", s)
	}
}

func TestInferType(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatalf("error creating tempfile for test")
	}

	defer os.Remove(tmpfile.Name()) // clean up

	s, err := Init(context.Background(), []*surfacerpb.SurfacerDef{
		{
			Surfacer: &surfacerpb.SurfacerDef_FileSurfacer{
				&fileconfigpb.SurfacerConf{
					FilePath: proto.String(tmpfile.Name()),
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(s) != 1 {
		t.Errorf("len(s)=%d, expected=1", len(s))
	}
	if _, ok := s[0].Surfacer.(*file.Surfacer); !ok {
		t.Errorf("Got surfacers for zero config: %v", s)
	}
}
