// Copyright 2017-2020 Google Inc.
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

Macros

Cloudprober configs support some macros to make configs construction easier:

 env
	Get the value of an environment variable.

	Example:

	probe {
	  name: "dns_google_jp"
	  type: DNS
	  targets {
	    host_names: "1.1.1.1"
	  }
	  dns_probe {
	    resolved_domain: "{{env "TEST_DOM"}}"
	  }
	}

	# Then run cloudprober as:
	TEST_DOM=google.co.jp ./cloudprober --config_file=cloudprober.cfg

 gceCustomMetadata
 	Get value of a GCE custom metadata key. It first looks for the given key in
	the instance's custom metadata and if it is not found there, it looks for it
	in the project's custom metaata.

	# Get load balancer IP from metadata.
	probe {
	  name: "http_lb"
	  type: HTTP
	  targets {
	    host_names: "{{gceCustomMetadata "lb_ip"}}"
	  }
	}

 extractSubstring
	Extract substring from a string using regex. Example use in config:

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

 mkMap
	Returns a map built from the arguments. It's useful as Go templates take only
	one argument. With this function, we can create a map of multiple values and
	pass it to a template. Example use in config:

	{{define "probeTmpl"}}
	probe {
	  type: {{.typ}}
	  name: "{{.name}}"
	  targets {
	    host_names: "www.google.com"
	  }
	}
	{{end}}

	{{template "probeTmpl" mkMap "typ" "PING" "name" "ping_google"}}
	{{template "probeTmpl" mkMap "typ" "HTTP" "name" "http_google"}}


 mkSlice
	Returns a slice consisting of the arguments. It can be used with the built-in
	'range' function to replicate text.


	{{with $regions := mkSlice "us=central1" "us-east1"}}
	{{range $_, $region := $regions}}

	probe {
	  name: "service-a-{{$region}}"
	  type: HTTP
	  targets {
	    host_names: "service-a.{{$region}}.corp.xx.com"
	  }
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

// ReadFromGCEMetadata returns the value of GCE custom metadata variables. To
// allow for instance level as project level variables, it looks for metadata
// variable in the following order:
// a. If the given key is set in the instance's custom metadata, its value is
//    returned.
// b. If (and only if), the key is not found in the step above, we look for
//    the same key in the project's custom metadata.
var ReadFromGCEMetadata = func(metadataKeyName string) (string, error) {
	val, err := metadata.InstanceAttributeValue(metadataKeyName)
	// If instance level config found, return
	if _, notFound := err.(metadata.NotDefinedError); !notFound {
		return val, err
	}
	// Check project level config next
	return metadata.ProjectAttributeValue(metadataKeyName)
}

// DefaultConfig returns the default config string.
func DefaultConfig() string {
	return proto.MarshalTextString(&configpb.ProberConfig{})
}

// ParseTemplate processes a config file as a Go text template.
func ParseTemplate(config string, sysVars map[string]string) (string, error) {
	return parseTemplate(config, sysVars, false)
}

// parseTemplate processes a config file as a Go text template.
func parseTemplate(config string, sysVars map[string]string, testMode bool) (string, error) {
	gceCustomMetadataFunc := func(v string) (string, error) {
		if testMode {
			return v + "-test-value", nil
		}

		// We return "undefined" if metadata variable is not defined.
		val, err := ReadFromGCEMetadata(v)
		if err != nil {
			if _, notFound := err.(metadata.NotDefinedError); notFound {
				return "undefined", nil
			}
			return "", err
		}
		return val, nil
	}

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

		"gceCustomMetadata": gceCustomMetadataFunc,

		// extractSubstring allows us to extract substring from a string using
		// regex matching groups.
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

// Parse processes a config file as a Go text template and parses it into a
// ProberConfig proto.
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

// ParseForTest processes a config file as a Go text template in test mode and
// parses it into a ProberConfig proto. This function is useful for testing
// configs.
func ParseForTest(config string, sysVars map[string]string) (*configpb.ProberConfig, error) {
	textConfig, err := parseTemplate(config, sysVars, true)
	if err != nil {
		return nil, err
	}
	cfg := &configpb.ProberConfig{}
	if err = proto.UnmarshalText(textConfig, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
