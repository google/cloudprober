// This binary implements a stand-alone ResourceDiscovery server.
package main

import (
	"context"
	"io/ioutil"
	"net"

	"flag"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/rds/server"
	configpb "github.com/google/cloudprober/rds/server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	config      = flag.String("config_file", "", "Config file (ServerConf)")
	addr        = flag.String("addr", ":0", "Port for the gRPC server")
	tlsCertFile = flag.String("tls_cert_file", ":0", "Port for the gRPC server")
	tlsKeyFile  = flag.String("tls_key_file", ":0", "Port for the gRPC server")
)

func main() {
	flag.Parse()

	c := &configpb.ServerConf{}

	// If we are given a config file, read it. If not, use defaults.
	if *config != "" {
		b, err := ioutil.ReadFile(*config)
		if err != nil {
			glog.Exit(err)
		}
		if err := proto.UnmarshalText(string(b), c); err != nil {
			glog.Exitf("Error while parsing config protobuf %s: Err: %v", string(b), err)
		}
	}

	grpcLn, err := net.Listen("tcp", *addr)
	if err != nil {
		glog.Exitf("error while creating listener for default gRPC server: %v", err)
	}

	// Create a gRPC server for RDS service.
	var serverOpts []grpc.ServerOption

	if *tlsCertFile != "" {
		creds, err := credentials.NewServerTLSFromFile(*tlsCertFile, *tlsKeyFile)
		if err != nil {
			glog.Exitf("error initializing gRPC server TLS credentials: %v", err)
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(serverOpts...)
	srv, err := server.New(context.Background(), c, nil, &logger.Logger{})
	if err != nil {
		glog.Exit(err)
	}
	srv.RegisterWithGRPC(grpcServer)

	grpcServer.Serve(grpcLn)
}
