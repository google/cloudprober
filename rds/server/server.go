// Copyright 2018 Google Inc.
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
Package server provides a ResourceDiscovery gRPC server implementation. It can
either be used a standalone server, using the binary in the "cmd" subdirectory,
or it can run as a part of the cloudprober binary.
*/
package server

import (
	"context"
	"fmt"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/rds/kubernetes"
	pb "github.com/google/cloudprober/rds/proto"
	spb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/gcp"
	configpb "github.com/google/cloudprober/rds/server/proto"
	"google.golang.org/grpc"
)

// Server implements a ResourceDiscovery gRPC server.
type Server struct {
	providers map[string]Provider
	l         *logger.Logger
}

// Provider is a resource provider, e.g. GCP provider.
type Provider interface {
	ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error)
}

// ListResources implements the ListResources method of the ResourceDiscovery
// service.
func (s *Server) ListResources(ctx context.Context, req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	p := s.providers[req.GetProvider()]
	if p == nil {
		return nil, fmt.Errorf("provider %s is not supported", req.GetProvider())
	}
	return p.ListResources(req)
}

func (s *Server) initProviders(c *configpb.ServerConf) error {
	var p Provider
	var err error
	for _, pc := range c.GetProvider() {
		id := pc.GetId()
		switch pc.Config.(type) {
		case *configpb.Provider_GcpConfig:
			if id == "" {
				id = gcp.DefaultProviderID
			}
			s.l.Infof("rds.server: adding GCP provider with id: %s", id)
			if p, err = gcp.New(pc.GetGcpConfig(), s.l); err != nil {
				return err
			}
		case *configpb.Provider_KubernetesConfig:
			if id == "" {
				id = kubernetes.DefaultProviderID
			}
			s.l.Infof("rds.server: adding Kubernetes provider with id: %s", id)
			if p, err = kubernetes.New(pc.GetKubernetesConfig(), s.l); err != nil {
				return err
			}
		}
		s.providers[id] = p
	}
	return nil
}

// New creates a new instance of the ResourceDiscovery Server and attaches it
// to the provided gRPC server.
func New(initCtx context.Context, c *configpb.ServerConf, providers map[string]Provider, l *logger.Logger) (*Server, error) {
	srv := &Server{
		providers: make(map[string]Provider),
		l:         l,
	}

	var err error

	if err = srv.initProviders(c); err != nil {
		return nil, err
	}

	// If providers are not nil. This option is mainly for testing.
	if providers != nil {
		for prefix, p := range providers {
			srv.providers[prefix] = p
		}
	}

	return srv, nil
}

// RegisterWithGRPC registers the RDS servers with the given gRPC server.
func (s *Server) RegisterWithGRPC(grpcServer *grpc.Server) {
	spb.RegisterResourceDiscoveryServer(grpcServer, s)
}
