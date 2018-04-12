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

*) mkSlice - mkSlice returns a slice consisting of arguments. Example use in config:

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

*/
package config

import (
	"bytes"
	"fmt"
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
		// mkSlice makes a slice from its arguments.
		"mkSlice": func(args ...interface{}) []interface{} {
			return args
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
