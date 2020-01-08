// Copyright 2019 Google Inc.
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
Package kubernetes implements a kubernetes resources provider for
ResourceDiscovery server.

It currently supports following kubernetes resources:
		Pods
		Endpoints
    Services

Kubernetes provider is configured through a protobuf based config file
(proto/config.proto). Example config:
{
  pods {}
}
*/
package kubernetes

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/kubernetes/proto"
	pb "github.com/google/cloudprober/rds/proto"
)

// DefaultProviderID is the povider id to use for this provider if a provider
// id is not configured explicitly.
const DefaultProviderID = "k8s"

// Provider implements a Kubernetes (K8s) provider for use with a
// ResourceDiscovery server.
type Provider struct {
	podsLister     *podsLister
	epLister       *epLister
	servicesLister *servicesLister
}

// kMetadata represents metadata for all Kubernetes resources.
type kMetadata struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

// ListResources returns the list of resources from the cache.
func (p *Provider) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	tok := strings.SplitN(req.GetResourcePath(), "/", 2)

	resType := tok[0]

	switch resType {
	case "pods":
		if p.podsLister == nil {
			return nil, errors.New("kubernetes: Pods lister not found")
		}
		resources, err := p.podsLister.listResources(req.GetFilter())
		return &pb.ListResourcesResponse{Resources: resources}, err
	case "endpoints":
		if p.epLister == nil {
			return nil, errors.New("kubernetes: Endpoints lister not found")
		}
		resources, err := p.epLister.listResources(req)
		return &pb.ListResourcesResponse{Resources: resources}, err
	case "services":
		if p.servicesLister == nil {
			return nil, errors.New("kubernetes: Services lister not found")
		}
		resources, err := p.servicesLister.listResources(req)
		return &pb.ListResourcesResponse{Resources: resources}, err
	default:
		return nil, fmt.Errorf("kubernetes: unsupported resource type: %s", resType)
	}
}

// New creates a Kubernetes (k8s) provider for RDS server, based on the
// provided config.
func New(c *configpb.ProviderConfig, l *logger.Logger) (*Provider, error) {
	client, err := newClient(c, l)
	if err != nil {
		return nil, fmt.Errorf("error while creating the kubernetes client: %v", err)
	}

	p := &Provider{}

	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second

	// Enable Pods lister if configured.
	if c.GetPods() != nil {
		podsLister, err := newPodsLister(c.GetPods(), c.GetNamespace(), reEvalInterval, client, l)
		if err != nil {
			return nil, err
		}
		p.podsLister = podsLister
	}

	// Enable Endpoints lister if configured.
	if c.GetEndpoints() != nil {
		epLister, err := newEndpointsLister(c.GetEndpoints(), c.GetNamespace(), reEvalInterval, client, l)
		if err != nil {
			return nil, err
		}
		p.epLister = epLister
	}

	// Enable Endpoints lister if configured.
	if c.GetServices() != nil {
		servicesLister, err := newServicesLister(c.GetServices(), c.GetNamespace(), reEvalInterval, client, l)
		if err != nil {
			return nil, err
		}
		p.servicesLister = servicesLister
	}

	return p, nil
}
