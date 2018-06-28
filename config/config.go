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

/*
Package config provides parser for cloudprober configs.

Example Usage:
	c, err := config.Parse(*configFile, sysvars.SysVars())

Parse processes a config file as a Go text template and parses it into a ProberConfig proto.
Config file is processed using the provided variable map (usually GCP metadata variables)
and some predefined macros.

Cloudprober configs support following macros:
*) env - get the value of an environment variable. A common use-case for this
  is using it inside a kubernetes cluster. Example:

	# Use an environment variable to set a
	probe {
	  name: "dns_google_jp"
	  type: DNS
	  targets {
	    host_names: "1.1.1.1"
	  }
	  dns_probe {
	    resolved_domain: "{{env "TEST_DOM"}}"
	  }
	  interval_msec: 5000  # 5s
	  timeout_msec: 1000   # 1s
	}
	# Then run cloudprober as:
	TEST_DOM=google.co.jp ./cloudprober --config_file=cloudprober.cfg

*) extractSubstring - extract substring from a string using regex. Example use in config:

	# Sharded VM-to-VM connectivity checks over internal IP
	# Instance name format: ig-<zone>-<shard>-<random-characters>, e.g. ig-asia-east1-a-00-ftx1
	{{$shard := .instance | extractSubstring "[^-]+-[^-]+-[^-]+-[^-]+-([^-]+)-.*" 1}}
	probe {
	  name: "vm-to-vm-{{$shard}}"
	  type: PING
	  targets {
	    gce_targets {
	      instances {}
            }
	    regex: "{{$targets}}"
          }
          run_on: "{{$run_on}}"
	}

*) mkMap - mkMap returns a map built from the arguments. It's useful as Go
  templates take only one argument. With this function, we can create a map of
  multiple values and pass it to a template. Example use in config:

	# Declare targets template that takes a map as an argument.
	{{define "targetsTmpl" -}}
	  rds_targets {
	    server_addr: "{{.rdsServer}}"

	    request {
	      provider: "gcp"
	      resource_path: "gce_instances/{{.project}}"
	      filter {
	        key: "name"
		regex: "{{.regex}}"
	      }
            }
          }
	{{- end}}

	probe {
	  ...
	  targets {
            {{template "targetsTmpl" mkMap "project" .project "rdsServer" "rds-server:9314" "regex" $targetRegex}}
          }
	  ...
	}

*) mkSlice - mkSlice returns a slice consisting of the arguments. Example use in config:

	# Sharded VM-to-VM connectivity checks over internal IP
	# Instance name format: ig-<zone>-<shard>-<random-characters>, e.g. ig-asia-east1-a-00-ftx1

	{{with $shards := mkSlice "00" "01" "02" "03"}}
	{{range $_, $shard := $shards}}
	{{$targets := printf "^ig-([^-]+-[^-]+-[^-]+)-%s-[^-]+$" $shard}}
	{{$run_on := printf "^ig-([^-]+-[^-]+-[^-]+)-%s-[^-.]+(|[.].*)$" $shard}}

	probe {
	  name: "vm-to-vm-{{$shard}}"
	  type: PING
	  targets {
	    gce_targets {
	      instances {}
	    }
	    regex: "{{$targets}}"
          }
	  run_on: "{{$run_on}}"
	}

	{{end}}
	{{end}}
*/
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"text/template"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/protobuf/proto"
	configpb "github.com/google/cloudprober/config/proto"
)

// ReadFromGCEMetadata reads the config from the GCE metadata. To allow for
// instance level as well as project-wide config, we look for the config in
// metadata in the following manner:
//	a. If cloudprober_config is set in the instance's custom metadata, its
//         value is returned.
//	b. If (and only if), the key is not found in the step above, we look for
//	   the same key in the project's custom metadata.
func ReadFromGCEMetadata(metadataKeyName string) (string, error) {
	config, err := metadata.Get("instance/attributes/" + metadataKeyName)
	// If instance level config found, return
	if _, notFound := err.(metadata.NotDefinedError); !notFound {
		return config, err
	}
	// Check project level config next
	return metadata.Get("project/attributes/" + metadataKeyName)
}

// DefaultConfig returns the default config string.
func DefaultConfig() string {
	return proto.MarshalTextString(&configpb.ProberConfig{})
}

// ParseTemplate processes a config file as a Go text template.
func ParseTemplate(config string, sysVars map[string]string) (string, error) {
	funcMap := map[string]interface{}{
		// env allows a user to lookup the value of a environment variable in
		// the configuration
		"env": func(key string) string {
			value, ok := os.LookupEnv(key)
			if !ok {
				return ""
			}
			return value
		},
		// extractSubstring allows us to extract substring from a string using regex
		// matching groups.
		"extractSubstring": func(re string, n int, s string) (string, error) {
			r, err := regexp.Compile(re)
			if err != nil {
				return "", err
			}
			matches := r.FindStringSubmatch(s)
			if len(matches) <= n {
				return "", fmt.Errorf("Match number %d not found. Regex: %s, String: %s", n, re, s)
			}
			return matches[n], nil
		},
		// mkMap makes a map from its argume
		"mkMap": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid mkMap call, need even number of args")
			}
			m := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("map keys must be strings")
				}
				m[key] = values[i+1]
			}
			return m, nil
		},
		// mkSlice makes a slice from its arguments.
		"mkSlice": func(args ...interface{}) []interface{} {
			return args
		},
	}
	configTmpl, err := template.New("cloudprober_cfg").Funcs(funcMap).Parse(config)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := configTmpl.Execute(&b, sysVars); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Parse processes a config file as a Go text template and parses it into a ProberConfig proto.
func Parse(config string, sysVars map[string]string) (*configpb.ProberConfig, error) {
	textConfig, err := ParseTemplate(config, sysVars)
	if err != nil {
		return nil, err
	}
	cfg := &configpb.ProberConfig{}
	if err = proto.UnmarshalText(textConfig, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
