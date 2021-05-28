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
	"github.com/google/cloudprober/surfacers/common/compress"
	"github.com/google/cloudprober/surfacers/common/options"
	configpb "github.com/google/cloudprober/surfacers/file/proto"
)

func TestWrite(t *testing.T) {
	testWrite(t, false)
}

func TestWriteWithCompression(t *testing.T) {
	testWrite(t, true)
}

func testWrite(t *testing.T, compressionEnabled bool) {
	t.Helper()

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

		s := &Surfacer{
			c: &configpb.SurfacerConf{
				FilePath:           proto.String(f.Name()),
				CompressionEnabled: proto.Bool(compressionEnabled),
			},
			opts: &options.Options{
				MetricsBufferSize: 1000,
			},
		}
		id := time.Now().UnixNano()
		err = s.init(context.Background(), id)
		if err != nil {
			t.Errorf("Unable to create a new file surfacer: %v\ntest description: %s", err, tt.description)
		}

		s.Write(context.Background(), tt.em)
		s.close()

		dat, err := ioutil.ReadFile(f.Name())
		if err != nil {
			t.Errorf("Unable to open test output file for reading: %v\ntest description: %s", err, tt.description)
		}

		expectedStr := fmt.Sprintf("%s %d %s\n", s.c.GetPrefix(), id, tt.em.String())
		if compressionEnabled {
			compressed, err := compress.Compress([]byte(expectedStr))
			if err != nil {
				t.Errorf("Unexpected error while compressing bytes: %v, Input: %s", err, expectedStr)
			}
			expectedStr = string(compressed) + "\n"
		}

		if diff := pretty.Compare(expectedStr, string(dat)); diff != "" {
			t.Errorf("Message written does not match expected output (-want +got):\n%s\ntest description: %s", diff, tt.description)
		}
	}
}
