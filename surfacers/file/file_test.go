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

package file

/*
These tests wait 100 milliseconds in between a write command and the read of the
file that it writes to to allow for the write to disk to complete. This could
lead to flakiness in the future.
*/

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"

	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/surfacers/file/proto"
)

func TestRun(t *testing.T) {

	tests := []struct {
		description string
		em          *metrics.EventMetrics
	}{
		{
			description: "file write of float data",
			em:          metrics.NewEventMetrics(time.Now()).AddMetric("float-test", metrics.NewInt(123456)),
		},
		{
			description: "file write of string data",
			em:          metrics.NewEventMetrics(time.Now()).AddMetric("string-test", metrics.NewString("test-string")),
		},
	}

	for _, tt := range tests {

		// Create temporary file so we don't accidentally overwrite
		// another file during test.
		f, err := ioutil.TempFile("", "file_test")
		if err != nil {
			t.Errorf("Unable to create a new file for testing: %v\ntest description: %s", err, tt.description)
		}
		defer os.Remove(f.Name())

		s := &FileSurfacer{
			c: &configpb.SurfacerConf{FilePath: proto.String(f.Name())},
		}
		id := time.Now().UnixNano()
		err = s.init(id)
		if err != nil {
			t.Errorf("Unable to create a new file surfacer: %v\ntest description: %s", err, tt.description)
		}

		s.Write(context.Background(), tt.em)

		// Sleep for 100 milliseconds to allow the go thread that is
		// performing the write to finish writing to the file before
		// we read (otherwise we will read too early and the file will
		// be blank)
		time.Sleep(100 * time.Millisecond)

		dat, err := ioutil.ReadFile(f.Name())
		if err != nil {
			t.Errorf("Unable to open test output file for reading: %v\ntest description: %s", err, tt.description)
		}
		if diff := pretty.Compare(fmt.Sprintf("%s %d %s\n", s.c.GetPrefix(), id, tt.em.String()), string(dat)); diff != "" {
			t.Errorf("Message written does not match expected output (-want +got):\n%s\ntest description: %s", diff, tt.description)
		}
	}
}
