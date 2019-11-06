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
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets/endpoint"
	configpb "github.com/google/cloudprober/targets/gce/proto"
	dnsRes "github.com/google/cloudprober/targets/resolver"

	"github.com/google/cloudprober/rds/client"
	clientconfigpb "github.com/google/cloudprober/rds/client/proto"
	"github.com/google/cloudprober/rds/gcp"
	rdspb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server"
	serverconfigpb "github.com/google/cloudprober/rds/server/proto"
)

var global struct {
	mu      sync.RWMutex
	servers map[string]*server.Server
}

// Targets interface is redefined here (originally defined in targets.go) to
// allow returning private gceResources.
type Targets interface {
	// List produces list of targets.
	List() []string
	// Resolve, given a target name and IP Version will return the IP address for
	// that target.
	Resolve(name string, ipVer int) (net.IP, error)
}

// gceResources encapsulates a set of GCE resources of a particular type. It's
// mostly a wrapper around RDS clients created during the initialization step.
type gceResources struct {
	c          *configpb.TargetsConf
	globalOpts *configpb.GlobalOptions
	l          *logger.Logger

	// RDS parameters
	resourceType string
	ipConfig     *rdspb.IPConfig
	filters      []*rdspb.Filter
	clients      map[string]*client.Client

	// DNS config
	r      *dnsRes.Resolver
	useDNS bool
}

func initRDSServer(resourceType, apiVersion string, projects []string, reEvalInterval int, l *logger.Logger) (*server.Server, error) {
	global.mu.Lock()
	defer global.mu.Unlock()

	if global.servers == nil {
		global.servers = make(map[string]*server.Server)
	}

	if global.servers[resourceType] != nil {
		return global.servers[resourceType], nil
	}

	pc := gcp.DefaultProviderConfig(projects, map[string]string{gcp.ResourceTypes.GCEInstances: ""}, reEvalInterval, apiVersion)
	srv, err := server.New(context.Background(), &serverconfigpb.ServerConf{Provider: []*serverconfigpb.Provider{pc}}, nil, l)
	if err != nil {
		return nil, err
	}

	global.servers[resourceType] = srv
	return srv, nil
}

func newRDSClient(req *rdspb.ListResourcesRequest, listResourcesFunc client.ListResourcesFunc, l *logger.Logger) (*client.Client, error) {
	c := &clientconfigpb.ClientConf{
		Request:   req,
		ReEvalSec: proto.Int32(10), // Refresh client every 10s.
	}

	client, err := client.New(c, listResourcesFunc, l)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (gr *gceResources) rdsRequest(project string) *rdspb.ListResourcesRequest {
	return &rdspb.ListResourcesRequest{
		Provider:     proto.String("gcp"),
		ResourcePath: proto.String(fmt.Sprintf("%s/%s", gr.resourceType, project)),
		Filter:       gr.filters,
		IpConfig:     gr.ipConfig,
	}
}

func (gr *gceResources) initClients(projects []string) error {
	srv, err := initRDSServer(gr.resourceType, gr.globalOpts.GetApiVersion(), projects, int(gr.globalOpts.GetReEvalSec()), gr.l)
	if err != nil {
		return err
	}

	for _, project := range projects {
		client, err := newRDSClient(gr.rdsRequest(project), srv.ListResources, gr.l)
		if err != nil {
			return err
		}
		gr.clients[project] = client
	}

	return nil
}

// List returns the list of target names.
func (gr *gceResources) List() []string {
	var names []string
	for _, client := range gr.clients {
		names = append(names, client.List()...)
	}
	return names
}

// ListEndpoints returns the list of GCE endpoints.
func (gr *gceResources) ListEndpoints() []endpoint.Endpoint {
	var ep []endpoint.Endpoint
	for _, client := range gr.clients {
		ep = append(ep, client.ListEndpoints()...)
	}
	return ep
}

// Resolve resolves the name into an IP address. Unless explicitly configured
// to use DNS, we use the RDS client to determine the resource IPs.
func (gr *gceResources) Resolve(name string, ipVer int) (net.IP, error) {
	if gr.useDNS {
		return gr.r.Resolve(name, ipVer)
	}

	var ip net.IP
	var err error

	for _, client := range gr.clients {
		ip, err = client.Resolve(name, ipVer)
		if err == nil {
			return ip, err
		}
	}

	return ip, err
}

// New is a helper function to unpack a Targets proto into a Targets interface.
func New(conf *configpb.TargetsConf, globalOpts *configpb.GlobalOptions, res *dnsRes.Resolver, l *logger.Logger) (Targets, error) {
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

	gr := &gceResources{
		c:          conf,
		ipConfig:   instancesIPConfig(conf.GetInstances()),
		clients:    make(map[string]*client.Client),
		globalOpts: globalOpts,
		l:          l,
	}

	switch conf.Type.(type) {
	case *configpb.TargetsConf_Instances:
		gr.resourceType = "gce_instances"
		// Verify that config is correct.
		if err := verifyInstancesConfig(conf.GetInstances(), res); err != nil {
			return nil, err
		}

		if len(conf.GetInstances().GetLabel()) > 0 {
			filters, err := parseLabels(conf.GetInstances().GetLabel())
			if err != nil {
				return nil, err
			}
			gr.filters = filters
		}
		return gr, gr.initClients(projects)

	case *configpb.TargetsConf_ForwardingRules:
		// TODO(manugarg): implement forwarding rules using gceResources like instances.go.
		frp, err := newForwardingRules(projects[0], globalOpts, conf.GetForwardingRules(), l)
		if err != nil {
			return nil, err
		}
		return frp, nil
	}

	return nil, errors.New("unknown GCE targets type")
}
