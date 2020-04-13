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
Package gcp implements a GCP (Google Compute Platform) resources provider for
ResourceDiscovery server.

See ResourceTypes variable for the list of supported resource types.

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
// Note that "rtc_variables" resource type is deprecated now and will soon be
// removed.
var ResourceTypes = struct {
	GCEInstances, ForwardingRules, RTCVariables, PubsubMessages string
}{
	"gce_instances",
	"forwarding_rules",
	"rtc_variables",
	"pubsub_messages",
}

var resourcePathTmpl = "<resource_type>[/<project-id>]"

type lister interface {
	listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error)
}

// Provider implements a GCP provider for a ResourceDiscovery server.
type Provider struct {
	localProject string

	listers map[string]map[string]lister
}

// ListResources returns the list of resources from the cache.
func (p *Provider) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	tok := strings.SplitN(req.GetResourcePath(), "/", 2)
	resType := tok[0]
	project := tok[1]

	var projectListers map[string]lister

	if project == "" && len(p.listers) == 1 {
		for _, pL := range p.listers {
			projectListers = pL
		}
	} else {
		projectListers = p.listers[project]
		if projectListers == nil {
			return nil, fmt.Errorf("no lister found for the project: %s", project)
		}
	}

	lr := projectListers[resType]
	if lr == nil {
		return nil, fmt.Errorf("unknown resource type: %s", resType)
	}

	resources, err := lr.listResources(req)
	return &pb.ListResourcesResponse{Resources: resources}, err
}

func initGCPProject(project string, c *configpb.ProviderConfig, l *logger.Logger) (map[string]lister, error) {
	projectLister := make(map[string]lister)

	// Enable GCE instances lister if configured.
	if c.GetGceInstances() != nil {
		lr, err := newGCEInstancesLister(project, c.GetApiVersion(), c.GetGceInstances(), l)
		if err != nil {
			return nil, err
		}
		projectLister[ResourceTypes.GCEInstances] = lr
	}

	// Enable forwarding rules lister if configured.
	if c.GetForwardingRules() != nil {
		lr, err := newForwardingRulesLister(project, c.GetApiVersion(), c.GetForwardingRules(), l)
		if err != nil {
			return nil, err
		}
		projectLister[ResourceTypes.ForwardingRules] = lr
	}

	// Enable RTC variables lister if configured.
	if c.GetPubsubMessages() != nil {
		lr, err := newPubSubMsgsLister(project, c.GetPubsubMessages(), l)
		if err != nil {
			return nil, err
		}
		projectLister[ResourceTypes.PubsubMessages] = lr
	}

	// Enable RTC variables lister if configured.
	if c.GetRtcVariables() != nil {
		lr, err := newRTCVariablesLister(project, c.GetApiVersion(), c.GetRtcVariables(), l)
		if err != nil {
			return nil, err
		}
		projectLister[ResourceTypes.RTCVariables] = lr
	}

	return projectLister, nil
}

// New creates a GCP provider for RDS server, based on the provided config.
func New(c *configpb.ProviderConfig, l *logger.Logger) (*Provider, error) {
	projects := c.GetProject()
	if len(projects) == 0 {
		if !metadata.OnGCE() {
			return nil, errors.New("rds.gcp.New(): project not configured and not running on GCE")
		}

		project, err := metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("rds.gcp.New(): error getting the local project ID on GCE: %v", err)
		}

		projects = append(projects, project)
	}

	p := &Provider{
		listers: make(map[string]map[string]lister),
	}

	for _, project := range projects {
		projectLister, err := initGCPProject(project, c, l)
		if err != nil {
			return nil, err
		}
		p.listers[project] = projectLister
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
				ReEvalSec: proto.Int32(int32(reEvalSec)),
			}

		case ResourceTypes.ForwardingRules:
			c.ForwardingRules = &configpb.ForwardingRules{
				ReEvalSec: proto.Int32(int32(reEvalSec)),
			}

		case ResourceTypes.RTCVariables:
			c.RtcVariables = &configpb.RTCVariables{
				RtcConfig: []*configpb.RTCVariables_RTCConfig{
					{
						Name:      proto.String(v),
						ReEvalSec: proto.Int32(int32(reEvalSec)),
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
