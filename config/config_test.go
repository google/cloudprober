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

package config

import (
	"fmt"
	"testing"

	"cloud.google.com/go/compute/metadata"
)

func TestParse(t *testing.T) {
	testConfig := `
{{ $shard := "ig-us-east1-a-02-afgx" | extractSubstring "[^-]+-[^-]+-[^-]+-[^-]+-([^-]+)-.*" 1 }}
probe {
  type: PING
  name: "vm-to-google-{{$shard}}-{{.region}}"
  targets {
    host_names: "www.google.com"
  }
  ping_probe {
    use_datagram_socket: true
  }
}
`
	c, err := Parse(testConfig, map[string]string{
		"region": "testRegion",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.GetProbe()) != 1 {
		t.Errorf("Didn't get correct number of probes. Got: %d, Expected: %d", len(c.GetProbe()), 1)
	}
	probeName := c.GetProbe()[0].GetName()
	expectedName := "vm-to-google-02-testRegion"
	if probeName != expectedName {
		t.Errorf("Incorrect probe name. Got: %s, Expected: %s", probeName, expectedName)
	}
}

func TestParseMap(t *testing.T) {
	testConfig := `
{{define "probeTmpl"}}
probe {
  type: {{.typ}}
  name: "{{.name}}"
  targets {
    host_names: "www.google.com"
  }
  ping_probe {
    use_datagram_socket: true
  }
}
{{end}}

{{template "probeTmpl" mkMap "typ" "PING" "name" "ping_google"}}
`
	c, err := Parse(testConfig, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.GetProbe()) != 1 {
		t.Errorf("Didn't get correct number of probes. Got: %d, Expected: %d", len(c.GetProbe()), 1)
	}
	probeName := c.GetProbe()[0].GetName()
	expectedName := "ping_google"
	if probeName != expectedName {
		t.Errorf("Incorrect probe name. Got: %s, Expected: %s", probeName, expectedName)
	}
}

func TestParseForTest(t *testing.T) {
	testConfig := `
probe {
  type: PING
  name: "{{gceCustomMetadata "google-probe-name"}}-{{gceCustomMetadata "cluster"}}"
  targets {
    host_names: "www.google.com"
  }
}
`
	ReadFromGCEMetadata = func(key string) (string, error) {
		if key == "google-probe-name" {
			return "google_dot_com_from", nil
		}
		if key == "cluster" {
			return "", metadata.NotDefinedError("not defined")
		}
		return "", fmt.Errorf("not-implemented")
	}

	c, err := Parse(testConfig, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.GetProbe()) != 1 {
		t.Errorf("Didn't get correct number of probes. Got: %d, Expected: %d", len(c.GetProbe()), 1)
	}
	probeName := c.GetProbe()[0].GetName()
	expectedName := "google_dot_com_from-undefined"
	if probeName != expectedName {
		t.Errorf("Incorrect probe name. Got: %s, Expected: %s", probeName, expectedName)
	}
}
