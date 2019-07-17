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
	"net"

	"github.com/google/cloudprober/config/runconfig"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	pb "github.com/google/cloudprober/targets/rds/proto"
	spb "github.com/google/cloudprober/targets/rds/proto"
	"github.com/google/cloudprober/targets/rds/server/gcp"
	configpb "github.com/google/cloudprober/targets/rds/server/proto"
	"google.golang.org/grpc"
)

// Server implements a ResourceDiscovery gRPC server.
type Server struct {
	providers  map[string]Provider
	ln         net.Listener
	grpcServer *grpc.Server
	l          *logger.Logger
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
		switch pc.Config.(type) {
		case *configpb.Provider_GcpConfig:
			if p, err = gcp.New(pc.GetGcpConfig(), s.l); err != nil {
				return err
			}
		}
		s.providers[pc.GetId()] = p
	}
	return nil
}

// Addr returns the server's address, if listener has been initialized.
func (s *Server) Addr() net.Addr {
	if s.ln == nil {
		return nil
	}
	return s.ln.Addr()
}

// Start starts the gRPC server. It returns only when the provided is canceled
// or server panics.
func (s *Server) Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) {
	// If running own GRPC server, start it now.
	if s.ln != nil {
		s.l.Infof("Starting gRPC server at: %s", s.ln.Addr().String())
		go func() {
			<-ctx.Done()
			s.l.Infof("Context canceled. Shutting down the gRPC server at: %s", s.ln.Addr().String())
			s.grpcServer.Stop()
		}()
		s.grpcServer.Serve(s.ln)
	}
}

// New creates a new instance of the ResourceDiscovery Server using the
// provided config and returns it. It also creates a net.Listener on the
// configured port so that we can catch port conflict errors early.
// TODO(manugarg): Now that we have a global gRPC server, change New() to take
// *grpc.Server as an argument and provide it at the time of RDS server
// initialization.
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

	if c.GetAddr() != "" {
		l.Warningf("Specifying \"addr\" field in RDS server config is deprecated now and will be removed after release v0.10.3. Use cloudprober config option \"grpc_port\" to configure the gRPC server.")
		srv.grpcServer = grpc.NewServer()
		if srv.ln, err = net.Listen("tcp", c.GetAddr()); err != nil {
			return nil, err
		}
	} else {
		srv.grpcServer = runconfig.DefaultGRPCServer()
	}

	if srv.grpcServer == nil {
		return nil, fmt.Errorf("no address specified for the gRPC server and no default gRPC server found")
	}

	spb.RegisterResourceDiscoveryServer(srv.grpcServer, srv)

	// Cleanup listener if initCtx is canceled.
	if srv.ln != nil {
		go func() {
			<-initCtx.Done()
			srv.ln.Close()
		}()
	}

	return srv, nil
}
