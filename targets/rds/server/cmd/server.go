// This binary implements a stand-alone ResourceDiscovery server.
package main

import (
	"context"
	"io/ioutil"

	"flag"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets/rds/server"
	configpb "github.com/google/cloudprober/targets/rds/server/proto"
)

var (
	config = flag.String("config_file", "", "Config file (ServerConf)")
	addr   = flag.String("addr", ":0", "Port for the gRPC server")
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

	if *addr != "" {
		c.Addr = addr
	}

	srv, err := server.New(context.Background(), c, nil, &logger.Logger{})
	if err != nil {
		glog.Exit(err)
	}
	srv.Start(context.Background(), nil)
}
