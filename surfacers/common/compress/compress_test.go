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

package compress

/*
These tests wait 100 milliseconds in between a write command and the read of the
file that it writes to to allow for the write to disk to complete. This could
lead to flakiness in the future.
*/

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/cloudprober/logger"
)

// Test that compress bytes is compressing bytes appropriately.
func TestCompressBytes(t *testing.T) {
	testString := "string that is about to be compressed"
	compressed, err := Compress([]byte(testString))
	if err != nil {
		t.Errorf("Got error while compressing the test string (%s): %v", testString, err)
	}
	// Verified that following is a good string using:
	// in="H4sIAAAAAAAA/youKcrMS1coyUgsUcgsVkhMyi8tUSjJV0hKVUjOzy0oSi0uTk0BBAAA//+2oNy3JQAAAA=="
	// echo -n $in | base64 -d | gunzip -c
	expectedCompressed := "H4sIAAAAAAAA/youKcrMS1coyUgsUcgsVkhMyi8tUSjJV0hKVUjOzy0oSi0uTk0BBAAA//+2oNy3JQAAAA=="
	if string(compressed) != expectedCompressed {
		t.Errorf("compressBytes(), got=%s, expected=%s", string(compressed), expectedCompressed)
	}
}

func TestCompressionBufferFlush(t *testing.T) {
	var outbuf bytes.Buffer

	c := CompressionBuffer{
		buf:      new(bytes.Buffer),
		l:        &logger.Logger{},
		callback: func(b []byte) { outbuf.Write(b) },
	}

	testStr := "test string\n"
	// We write without newline as new line is added by WriteLineToBuffer.
	c.WriteLineToBuffer(strings.TrimSpace(testStr))

	if c.buf.Len() == 0 || c.lines == 0 {
		t.Errorf("compressionBuffer unexpectedly empty")
	}
	if !bytes.Equal(c.buf.Bytes(), []byte(testStr)) {
		t.Errorf("Buffer: %v, expected: %v", c.buf.Bytes(), []byte(testStr))
	}

	c.compressAndCallback()

	if c.buf.Len() != 0 && c.lines != 0 {
		t.Errorf("flush() didn't empty the compressionBuffer")
	}

	b, _ := Compress([]byte(testStr))
	if !bytes.Equal(outbuf.Bytes(), b) {
		t.Errorf("Got compressed data: %v, expected: %v", outbuf.Bytes(), b)
	}
}
