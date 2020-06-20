// Copyright 2020 Google Inc.
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

package iputils

import (
	"net"
	"testing"
)

func TestIPVersion(t *testing.T) {
	rows := []struct {
		ip    string
		ipVer int
	}{
		{"1.1.1.1", 4},
		{"::1", 6},
	}

	for _, r := range rows {
		ipVer := IPVersion(net.ParseIP(r.ip))

		if ipVer != r.ipVer {
			t.Errorf("Unexpected IPVersion want=%d, got=%d", r.ipVer, ipVer)
		}
	}
}
