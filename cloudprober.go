// Copyright 2017-2019 The Cloudprober Authors.
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
Package cloudprober provides a prober for running a set of probes.

Cloudprober takes in a config proto which dictates what probes should be created
with what configuration, and manages the asynchronous fan-in/fan-out of the
metrics data from these probes.
*/
package cloudprober

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/common/tlsconfig"
	"github.com/google/cloudprober/config"
	configpb "github.com/google/cloudprober/config/proto"
	"github.com/google/cloudprober/config/runconfig"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/prober"
	"github.com/google/cloudprober/probes"
	"github.com/google/cloudprober/servers"
	"github.com/google/cloudprober/surfacers"
	"github.com/google/cloudprober/sysvars"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/credentials"
)

const (
	sysvarsModuleName = "sysvars"
)

// Constants defining the default server host and port.
const (
	DefaultServerHost = ""
	DefaultServerPort = 9313
	ServerHostEnvVar  = "CLOUDPROBER_HOST"
	ServerPortEnvVar  = "CLOUDPROBER_PORT"
)

// Global prober.Prober instance protected by a mutex.
var cloudProber struct {
	prober          *prober.Prober
	defaultServerLn net.Listener
	defaultGRPCLn   net.Listener
	textConfig      string
	config          *configpb.ProberConfig
	cancelInitCtx   context.CancelFunc
	sync.Mutex
}

func getServerHost(c *configpb.ProberConfig) string {
	serverHost := c.GetHost()
	if serverHost == "" {
		serverHost = DefaultServerHost
		// If ServerHostEnvVar is defined, it will override the default
		// server host.
		if host := os.Getenv(ServerHostEnvVar); host != "" {
			serverHost = host
		}
	}
	return serverHost
}

func getDefaultServerPort(c *configpb.ProberConfig, l *logger.Logger) (int, error) {
	if c.GetPort() != 0 {
		return int(c.GetPort()), nil
	}

	// If ServerPortEnvVar is defined, it will override the default
	// server port.
	portStr := os.Getenv(ServerPortEnvVar)
	if portStr == "" {
		return DefaultServerPort, nil
	}

	if strings.HasPrefix(portStr, "tcp://") {
		l.Warningf("%s environment variable likely set by Kubernetes (to %s), ignoring it", ServerPortEnvVar, portStr)
		return DefaultServerPort, nil
	}

	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse default port from the env var: %s=%s", ServerPortEnvVar, portStr)
	}

	return int(port), nil
}

func initDefaultServer(c *configpb.ProberConfig, l *logger.Logger) (net.Listener, error) {
	serverHost := getServerHost(c)
	serverPort, err := getDefaultServerPort(c, l)

	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", serverHost, serverPort))
	if err != nil {
		return nil, fmt.Errorf("error while creating listener for default HTTP server: %v", err)
	}

	return ln, nil
}

// InitFromConfig initializes Cloudprober using the provided config.
func InitFromConfig(configFile string) error {
	// Return immediately if prober is already initialized.
	cloudProber.Lock()
	defer cloudProber.Unlock()

	if cloudProber.prober != nil {
		return nil
	}

	// Initialize sysvars module
	l, err := logger.NewCloudproberLog(sysvarsModuleName)
	if err != nil {
		return err
	}

	if err := sysvars.Init(l, nil); err != nil {
		return err
	}

	configStr, err := config.ParseTemplate(configFile, sysvars.Vars())
	if err != nil {
		return err
	}

	cfg := &configpb.ProberConfig{}
	if err := proto.UnmarshalText(configStr, cfg); err != nil {
		return err
	}

	globalLogger, err := logger.NewCloudproberLog("global")
	if err != nil {
		return fmt.Errorf("error in initializing global logger: %v", err)
	}

	// Start default HTTP server. It's used for profile handlers and
	// prometheus exporter.
	ln, err := initDefaultServer(cfg, l)
	if err != nil {
		return err
	}

	var grpcLn net.Listener
	if cfg.GetGrpcPort() != 0 {
		serverHost := getServerHost(cfg)

		grpcLn, err = net.Listen("tcp", fmt.Sprintf("%s:%d", serverHost, cfg.GetGrpcPort()))
		if err != nil {
			return fmt.Errorf("error while creating listener for default gRPC server: %v", err)
		}

		// Create the default gRPC server now, so that other modules can register
		// their services with it in the prober.Init() phase.
		var serverOpts []grpc.ServerOption

		if cfg.GetGrpcTlsConfig() != nil {
			tlsConfig := &tls.Config{}
			if err := tlsconfig.UpdateTLSConfig(tlsConfig, cfg.GetGrpcTlsConfig(), true); err != nil {
				return err
			}
			serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		}

		s := grpc.NewServer(serverOpts...)
		// register channelz service to the default grpc server port
		service.RegisterChannelzServiceToServer(s)
		runconfig.SetDefaultGRPCServer(s)
	}

	pr := &prober.Prober{}

	// initCtx is used to clean up in case of partial initialization failures. For
	// example, user-configured servers open listeners during initialization and
	// if initialization fails at a later stage, say in probers or surfacers,
	// pr.Init returns an error and we cancel the initCtx, which makes servers
	// close their listeners.
	// TODO(manugarg): Plumb init context from cmd/cloudprober.
	initCtx, cancelFunc := context.WithCancel(context.TODO())
	if err := pr.Init(initCtx, cfg, globalLogger); err != nil {
		cancelFunc()
		ln.Close()
		return err
	}

	cloudProber.prober = pr
	cloudProber.config = cfg
	cloudProber.textConfig = configStr
	cloudProber.defaultServerLn = ln
	cloudProber.defaultGRPCLn = grpcLn
	cloudProber.cancelInitCtx = cancelFunc
	return nil
}

// Start starts a previously initialized Cloudprober.
func Start(ctx context.Context) {
	cloudProber.Lock()
	defer cloudProber.Unlock()

	// Default servers
	srv := &http.Server{}
	grpcSrv := runconfig.DefaultGRPCServer()

	// Set up a goroutine to cleanup if context ends.
	go func() {
		<-ctx.Done()
		srv.Close() // This will close the listener as well.
		if grpcSrv != nil {
			grpcSrv.Stop()
		}
		cloudProber.cancelInitCtx()
	}()

	go srv.Serve(cloudProber.defaultServerLn)
	if grpcSrv != nil && cloudProber.defaultGRPCLn != nil {
		go grpcSrv.Serve(cloudProber.defaultGRPCLn)
	}

	if cloudProber.prober == nil {
		panic("Prober is not initialized. Did you call cloudprober.InitFromConfig first?")
	}

	cloudProber.prober.Start(ctx)
}

// GetConfig returns the prober config.
func GetConfig() *configpb.ProberConfig {
	cloudProber.Lock()
	defer cloudProber.Unlock()
	return cloudProber.config
}

// GetTextConfig returns the prober config in text proto format.
func GetTextConfig() string {
	cloudProber.Lock()
	defer cloudProber.Unlock()
	return cloudProber.textConfig
}

// GetInfo returns information on all the probes, servers and surfacers.
func GetInfo() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {
	cloudProber.Lock()
	defer cloudProber.Unlock()
	return cloudProber.prober.Probes, cloudProber.prober.Surfacers, cloudProber.prober.Servers
}
