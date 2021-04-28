// Copyright 2017-2019 The Cloudprober Authors.
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

package probeutils

import (
	"reflect"
	"testing"
)

func TestPayloadVerification(t *testing.T) {
	testBytes := []byte("test bytes")

	// Verify that for the larger payload sizes we get replicas of the same
	// bytes.
	for _, size := range []int{4, 256, 999, 2048, 4 * len(testBytes)} {
		payload := make([]byte, size)

		PatternPayload(payload, testBytes)

		var expectedBuf []byte
		for i := 0; i < size/len(testBytes); i++ {
			expectedBuf = append(expectedBuf, testBytes...)
		}
		// Remaining bytes
		expectedBuf = append(expectedBuf, testBytes[:size-len(expectedBuf)]...)
		if !reflect.DeepEqual(payload, expectedBuf) {
			t.Errorf("Bytes array:\n%o\n\nExpected:\n%o", payload, expectedBuf)
		}

		// Verify payload.
		err := VerifyPayloadPattern(payload, testBytes)
		if err != nil {
			t.Errorf("Data verification error: %v", err)
		}
	}
}

func benchmarkVerifyPayloadPattern(size int, b *testing.B) {
	testBytes := []byte("test bytes")
	payload := make([]byte, size)
	PatternPayload(payload, testBytes)

	for n := 0; n < b.N; n++ {
		VerifyPayloadPattern(payload, testBytes)
	}
}

func BenchmarkVerifyPayloadPattern56(b *testing.B)   { benchmarkVerifyPayloadPattern(56, b) }
func BenchmarkVerifyPayloadPattern256(b *testing.B)  { benchmarkVerifyPayloadPattern(256, b) }
func BenchmarkVerifyPayloadPattern1999(b *testing.B) { benchmarkVerifyPayloadPattern(1999, b) }
func BenchmarkVerifyPayloadPattern9999(b *testing.B) { benchmarkVerifyPayloadPattern(9999, b) }
