// Copyright 2017-2018 Google Inc.
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
Package gcp implements a GCP (Google Compute Platform) resources provider for
ResourceDiscovery server.

It currently supports following GCP resources:
		GCE Instances (gce_instance)

GCP provider is configured through a protobuf based config file
(proto/config.proto). Example config:
{
  project_id: 'test-project-1'
  project_id: 'test-project-2'
  gce_instances {}
}
*/
package gcp

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/google/cloudprober/logger"
	pb "github.com/google/cloudprober/targets/rds/proto"
	configpb "github.com/google/cloudprober/targets/rds/server/gcp/proto"
)

// Provider implements a GCP provider for a ResourceDiscovery server.
type Provider struct {
	gceInstances map[string]*gceInstancesLister
}

// ListResources returns the list of resources from the cache.
func (p *Provider) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	tok := strings.SplitN(req.GetResourcePath(), "/", 2)
	if len(tok) != 2 {
		return nil, fmt.Errorf("%s is not a valid GCP resource path", req.GetResourcePath())
	}
	resType := tok[0]
	project := tok[1]

	switch resType {
	case "gce_instances":
		gil := p.gceInstances[project]
		if gil == nil {
			return nil, fmt.Errorf("gcp: lister for the project %s not found", project)
		}
		return &pb.ListResourcesResponse{
			Resources: gil.listResources(req.GetFilter(), req.GetIpConfig()),
		}, nil
	default:
		return nil, fmt.Errorf("gcp: unsupported resource type: %s", resType)
	}
}

// New creates a GCP provider for RDS server, based on the provided config.
func New(c *configpb.ProviderConfig, l *logger.Logger) (*Provider, error) {
	projects := c.GetProject()
	if len(projects) == 0 {
		if !metadata.OnGCE() {
			return nil, errors.New("rds.gcp.New(): project is required for GCP resources when not running on GCE")
		}
		proj, err := metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("rds.gcp.New(): error getting local project ID: %v", err)
		}
		projects = append(projects, proj)
	}

	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second

	p := &Provider{
		gceInstances: make(map[string]*gceInstancesLister),
	}

	// Enable GCE instances lister if configured.
	if c.GetGceInstances() != nil {
		for _, project := range projects {
			gil, err := newGCEInstancesLister(project, c.GetApiVersion(), reEvalInterval, c.GetGceInstances(), l)
			if err != nil {
				return nil, err
			}
			p.gceInstances[project] = gil
		}
	}
	// Enable regional forwarding lister if configured.
	// TODO(manugarg): implement this.
	if c.GetRegionalForwardingRules() != nil {
		return nil, errors.New("regional forwarding rules are not supported yet")
	}
	return p, nil
}
