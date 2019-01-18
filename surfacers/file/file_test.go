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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
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

		s := &FileSurfacer{
			c: &configpb.SurfacerConf{
				FilePath:           proto.String(f.Name()),
				CompressionEnabled: proto.Bool(compressionEnabled),
			},
		}
		id := time.Now().UnixNano()
		err = s.init(context.Background(), id)
		if err != nil {
			t.Errorf("Unable to create a new file surfacer: %v\ntest description: %s", err, tt.description)
		}

		s.Write(context.Background(), tt.em)

		// Sleep for 100 milliseconds to allow the go thread that is
		// performing the write to finish writing to the file before
		// we read (otherwise we will read too early and the file will
		// be blank)
		// time.Sleep(100 * time.Millisecond)
		time.Sleep(compressionBufferFlushInterval + time.Second)

		dat, err := ioutil.ReadFile(f.Name())
		if err != nil {
			t.Errorf("Unable to open test output file for reading: %v\ntest description: %s", err, tt.description)
		}

		expectedStr := fmt.Sprintf("%s %d %s\n", s.c.GetPrefix(), id, tt.em.String())
		if compressionEnabled {
			compressed, err := compressBytes([]byte(expectedStr))
			if err != nil {
				t.Errorf("Unexpected error while compressing bytes: %v, Input: %s", err, expectedStr)
			}
			expectedStr = compressed + "\n"
		}

		if diff := pretty.Compare(expectedStr, string(dat)); diff != "" {
			t.Errorf("Message written does not match expected output (-want +got):\n%s\ntest description: %s", diff, tt.description)
		}
	}
}

// Test that compress bytes is compressing bytes appropriately.
func TestCompressBytes(t *testing.T) {
	testString := "string that is about to be compressed"
	compressed, err := compressBytes([]byte(testString))
	if err != nil {
		t.Errorf("Got error while compressing the test string (%s): %v", testString, err)
	}
	// Verified that following is a good string using:
	// in="H4sIAAAAAAAA/youKcrMS1coyUgsUcgsVkhMyi8tUSjJV0hKVUjOzy0oSi0uTk0BBAAA//+2oNy3JQAAAA=="
	// echo -n $in | base64 -d | gunzip -c
	expectedCompressed := "H4sIAAAAAAAA/youKcrMS1coyUgsUcgsVkhMyi8tUSjJV0hKVUjOzy0oSi0uTk0BBAAA//+2oNy3JQAAAA=="
	if compressed != expectedCompressed {
		t.Errorf("compressBytes(), got=%s, expected=%s", compressed, expectedCompressed)
	}
}

func TestCompressionBufferFlush(t *testing.T) {
	c := &compressionBuffer{
		buf:     new(bytes.Buffer),
		outChan: make(chan string, 1000),
		l:       &logger.Logger{},
	}

	// Write a test line to the buffer.
	c.writeLine("test string")

	if c.buf.Len() == 0 || c.lines == 0 {
		t.Errorf("compressionBuffer unexpected empty")
	}

	c.flush()

	if c.buf.Len() != 0 && c.lines != 0 {
		t.Errorf("flush() didn't empty the compressionBuffer")
	}
}
