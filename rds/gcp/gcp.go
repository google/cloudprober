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

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/gcp/proto"
	pb "github.com/google/cloudprober/rds/proto"
	serverconfigpb "github.com/google/cloudprober/rds/server/proto"
)

// DefaultProviderID is the povider id to use for this provider if a provider
// id is not configured explicitly.
const DefaultProviderID = "gcp"

// ResourceTypes declares resource types supported by the GCP provider.
var ResourceTypes = struct {
	GCEInstances, RTCVariables, PubsubMessages string
}{
	"gce_instances",
	"rtc_variables",
	"pubsub_messages",
}

// Provider implements a GCP provider for a ResourceDiscovery server.
type Provider struct {
	gceInstances map[string]*gceInstancesLister
	rtcVariables map[string]*rtcVariablesLister
	pubsubMsgs   map[string]*pubsubMsgsLister
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
	case ResourceTypes.GCEInstances:
		gil := p.gceInstances[project]
		if gil == nil {
			return nil, fmt.Errorf("gcp: GCE instances lister for the project %s not found", project)
		}
		resources, err := gil.listResources(req.GetFilter(), req.GetIpConfig())
		return &pb.ListResourcesResponse{Resources: resources}, err
	case ResourceTypes.RTCVariables:
		rvl := p.rtcVariables[project]
		if rvl == nil {
			return nil, fmt.Errorf("gcp: RTC variables lister for the project %s not found", project)
		}
		resources, err := rvl.listResources(req.GetFilter())
		return &pb.ListResourcesResponse{Resources: resources}, err
	case ResourceTypes.PubsubMessages:
		lister := p.pubsubMsgs[project]
		if lister == nil {
			return nil, fmt.Errorf("gcp: Pub/Sub messages lister for the project %s not found", project)
		}
		resources, err := lister.listResources(req.GetFilter())
		return &pb.ListResourcesResponse{Resources: resources}, err
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

	p := &Provider{
		gceInstances: make(map[string]*gceInstancesLister),
		rtcVariables: make(map[string]*rtcVariablesLister),
		pubsubMsgs:   make(map[string]*pubsubMsgsLister),
	}

	// Enable GCE instances lister if configured.
	if c.GetGceInstances() != nil {
		for _, project := range projects {
			gil, err := newGCEInstancesLister(project, c.GetApiVersion(), c.GetGceInstances(), l)
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

	// Enable RTC variables lister if configured.
	if c.GetPubsubMessages() != nil {
		for _, project := range projects {
			lister, err := newPubSubMsgsLister(project, c.GetPubsubMessages(), l)
			if err != nil {
				return nil, err
			}
			p.pubsubMsgs[project] = lister
		}
	}

	// Enable RTC variables lister if configured.
	if c.GetRtcVariables() != nil {
		for _, project := range projects {
			rvl, err := newRTCVariablesLister(project, c.GetApiVersion(), c.GetRtcVariables(), l)
			if err != nil {
				return nil, err
			}
			p.rtcVariables[project] = rvl
		}
	}
	return p, nil
}

// DefaultProviderConfig is a convenience function that builds and returns a
// basic GCP provider config based on the given parameters.
func DefaultProviderConfig(projects []string, resTypes map[string]string, reEvalSec int, apiVersion string) *serverconfigpb.Provider {
	c := &configpb.ProviderConfig{
		Project:    projects,
		ApiVersion: proto.String(apiVersion),
	}

	for k, v := range resTypes {
		switch k {
		case ResourceTypes.GCEInstances:
			c.GceInstances = &configpb.GCEInstances{
				ReEvalSec: proto.Int(reEvalSec),
			}
		case ResourceTypes.RTCVariables:
			c.RtcVariables = &configpb.RTCVariables{
				RtcConfig: []*configpb.RTCVariables_RTCConfig{
					{
						Name:      proto.String(v),
						ReEvalSec: proto.Int(reEvalSec),
					},
				},
			}
		case ResourceTypes.PubsubMessages:
			c.PubsubMessages = &configpb.PubSubMessages{
				Subscription: []*configpb.PubSubMessages_Subscription{
					{
						TopicName: proto.String(v),
					},
				},
			}
		}
	}

	return &serverconfigpb.Provider{
		Id:     proto.String(DefaultProviderID),
		Config: &serverconfigpb.Provider_GcpConfig{GcpConfig: c},
	}
}
