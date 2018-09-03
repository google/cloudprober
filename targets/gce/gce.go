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

// Package gce implements Google Compute Engine (GCE) targets for Cloudprober.
//
// It currently supports following GCE targets:
//	Instances
//	Forwarding Rules (only regional currently)
//
// Targets are configured through a config file, based on the protobuf defined
// in the config.proto file in the same directory. Example config:
//
// All instances matching a certain regex:
//	targets {
//	  gce_targets {
//	    instances {}
//	  }
//	  regex: "ig-proxy-.*"
//	}
//
// Public IP of all instances matching a certain regex:
//	targets {
//	  gce_targets {
//	    instances {
//	      public_ip: true
//	    }
//	  }
//	  regex: "ig-proxy-.*"
//	}
//
// All forwarding rules in the local region:
//	targets {
//	  gce_targets {
//	    forwarding_rules {}
//	  }
//	}
package gce

import (
	"errors"
	"fmt"
	"net"

	"cloud.google.com/go/compute/metadata"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/targets/gce/proto"
	dnsRes "github.com/google/cloudprober/targets/resolver"
)

// Targets are able to list and resolve targets with their List and Resolve
// methods.  A single instance of Targets represents a specific listing method
// --- if multiple sets of resources need to be listed/resolved, a separate
// instance of Targets will be needed.
type Targets interface {
	// List produces list of targets.
	List() []string
	// Resolve, given a target name and IP Version will return the IP address for
	// that target.
	Resolve(name string, ipVer int) (net.IP, error)
}

// New is a helper function to unpack a Targets proto into a Targets interface.
func New(conf *configpb.TargetsConf, opts *configpb.GlobalOptions, res *dnsRes.Resolver, log *logger.Logger) (t Targets, err error) {
	projects := conf.GetProject()
	if projects == nil {
		if !metadata.OnGCE() {
			return nil, errors.New("targets.gce.New(): project is required for GCE targets when not running on GCE")
		}
		currentProj, err := metadata.ProjectID()
		projects = append([]string{}, currentProj)
		if err != nil {
			return nil, fmt.Errorf("targets.gce.New(): Error getting project ID: %v", err)
		}
	}
	switch conf.Type.(type) {
	case *configpb.TargetsConf_Instances:
		t, err = newInstances(projects, opts, conf.GetInstances(), res, log)
	case *configpb.TargetsConf_ForwardingRules:
		t, err = newForwardingRules(projects[0], opts, log)
	default:
		err = errors.New("unknown GCE targets type")
	}
	return
}
