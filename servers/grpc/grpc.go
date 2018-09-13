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

// Package grpc provides a simple gRPC server that acts as a probe target.
package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/reflection"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/servers/grpc/proto"
	grpcpb "github.com/google/cloudprober/servers/grpc/proto"
	spb "github.com/google/cloudprober/servers/grpc/proto"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Server implements a gRPCServer.
type Server struct {
	c         *configpb.ServerConf
	ln        net.Listener
	grpcSrv   *grpc.Server
	healthSrv *health.Server
	l         *logger.Logger
	startTime time.Time
}

// Echo reflects back the incoming message.
func (s *Server) Echo(ctx context.Context, req *spb.EchoMessage) (*spb.EchoMessage, error) {
	return req, nil
}

// ServerStatus returns the current server status.
func (s *Server) ServerStatus(ctx context.Context, req *spb.StatusRequest) (*spb.StatusResponse, error) {
	return &spb.StatusResponse{
		UptimeUs: proto.Int64(time.Since(s.startTime).Nanoseconds() / 1000),
	}, nil
}

// New returns a Server.
func New(initCtx context.Context, c *configpb.ServerConf, l *logger.Logger) (*Server, error) {
	grpcSrv := grpc.NewServer()
	healthSrv := health.NewServer()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", c.GetPort()))
	if err != nil {
		return nil, err
	}
	// Cleanup listener if initCtx is canceled.
	go func() {
		<-initCtx.Done()
		ln.Close()
	}()

	srv := &Server{
		c:         c,
		l:         l,
		ln:        ln,
		grpcSrv:   grpcSrv,
		healthSrv: healthSrv,
		startTime: time.Now(),
	}

	grpcpb.RegisterProberServer(grpcSrv, srv)
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)
	return srv, nil
}

// Start starts the gRPC server and serves requests until the context is
// canceled or the gRPC server panics.
func (s *Server) Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error {
	s.l.Infof("Starting gRPC server at %s", s.ln.Addr().String())
	go func() {
		<-ctx.Done()
		s.l.Infof("Context canceled. Shutting down the gRPC server at: %s", s.ln.Addr().String())
		for svc := range s.grpcSrv.GetServiceInfo() {
			s.healthSrv.SetServingStatus(svc, healthpb.HealthCheckResponse_NOT_SERVING)
		}
		s.grpcSrv.Stop()
	}()
	for si := range s.grpcSrv.GetServiceInfo() {
		s.healthSrv.SetServingStatus(si, healthpb.HealthCheckResponse_SERVING)
	}
	if s.c.GetEnableReflection() {
		s.l.Infof("Enabling reflection for gRPC server at %s", s.ln.Addr().String())
		reflection.Register(s.grpcSrv)
	}
	return s.grpcSrv.Serve(s.ln)
}
