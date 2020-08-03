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

/*
Package probeutils implements utilities that are shared across multiple probe
types.
*/
package probeutils

import (
	"bytes"
	"fmt"
)

// PatternPayload builds a payload that can be verified using VerifyPayloadPattern.
// It repeats the pattern to fill the payload []byte slice. Last remaining
// bytes (len(payload) mod patternSize) are left unpopulated (hence set to 0
// bytes).
func PatternPayload(payload, pattern []byte) {
	patternSize := len(pattern)
	for i := 0; i < len(payload); i += patternSize {
		copy(payload[i:], pattern)
	}
}

// VerifyPayloadPattern verifies the payload built using PatternPayload.
func VerifyPayloadPattern(payload, pattern []byte) error {
	patternSize := len(pattern)
	nReplica := len(payload) / patternSize

	for i := 0; i < nReplica; i++ {
		bN := payload[0:patternSize]    // Next pattern sized bytes
		payload = payload[patternSize:] // Shift payload for next iteration

		if !bytes.Equal(bN, pattern) {
			return fmt.Errorf("bytes are not in the expected format. payload[%d-Replica]=%v, pattern=%v", i, bN, pattern)
		}
	}

	if !bytes.Equal(payload, pattern[:len(payload)]) {
		return fmt.Errorf("last %d bytes are not in the expected format. payload=%v, expected=%v", len(payload), payload, pattern[:len(payload)])
	}

	return nil
}
