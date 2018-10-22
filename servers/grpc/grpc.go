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
	c           *configpb.ServerConf
	ln          net.Listener
	ready       chan bool
	grpcSrv     *grpc.Server
	healthSrv   *health.Server
	l           *logger.Logger
	startTime   time.Time
	injectedSrv bool
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
	return &Server{
		c:     c,
		l:     l,
		ready: make(chan bool),
	}, nil
}

func (s *Server) setupDefaultServer(ctx context.Context) error {
	grpcSrv := grpc.NewServer()
	healthSrv := health.NewServer()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.c.GetPort()))
	if err != nil {
		return err
	}
	// Cleanup listener if ctx is canceled.
	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	s.ln = ln
	s.grpcSrv = grpcSrv
	s.healthSrv = healthSrv
	s.startTime = time.Now()

	grpcpb.RegisterProberServer(grpcSrv, s)
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)
	close(s.ready)

	return nil
}

// InjectGRPCServer allows caller to attach an externally configured gRPC
// server to implement and serve Cloudprober's services. Caller has to
// start, serve and stop the gRPC server.
func (s *Server) InjectGRPCServer(grpcServer *grpc.Server) error {
	if s.grpcSrv != nil {
		return fmt.Errorf("gRPC server already attached: %v(injected=%v)", s.grpcSrv, s.injectedSrv)
	}
	s.grpcSrv = grpcServer
	s.injectedSrv = true
	s.startTime = time.Now()
	grpcpb.RegisterProberServer(grpcServer, s)
	return nil
}

// Start starts the gRPC server and serves requests until the context is
// canceled or the gRPC server panics.
func (s *Server) Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error {
	if s.injectedSrv {
		// Nothing to do as caller owns server. Wait till context is done.
		<-ctx.Done()
		return nil
	}
	if err := s.setupDefaultServer(ctx); err != nil {
		return err
	}
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
