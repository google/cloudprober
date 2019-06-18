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

// This binary implements a Cloudprober gRPC client. This binary is to only
// demonstrate how cloudprober can be programmed dynamically.
//
// go run ./cmd/client.go --server localhost:9314 --add_probe newprobe.cfg
// go run ./cmd/client.go --server localhost:9314 --rm_probe newprobe
package main

import (
	"context"
	"io/ioutil"

	"flag"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	pb "github.com/google/cloudprober/prober/proto"
	spb "github.com/google/cloudprober/prober/proto"
	configpb "github.com/google/cloudprober/probes/proto"
	"google.golang.org/grpc"
)

var (
	server   = flag.String("server", "", "gRPC server address")
	addProbe = flag.String("add_probe", "", "Path to probe config to add")
	rmProbe  = flag.String("rm_probe", "", "Probe name to remove")
)

func main() {
	flag.Parse()

	conn, err := grpc.Dial(*server, grpc.WithInsecure())
	if err != nil {
		glog.Fatal(err)
	}

	client := spb.NewCloudproberClient(conn)

	if *addProbe != "" && *rmProbe != "" {
		glog.Fatal("Options --add_probe and --rm_probe cannot be specified at the same time")
	}

	if *rmProbe != "" {
		_, err := client.RemoveProbe(context.Background(), &pb.RemoveProbeRequest{ProbeName: rmProbe})
		if err != nil {
			glog.Exit(err)
		}
	}

	if *addProbe != "" {
		b, err := ioutil.ReadFile(*addProbe)
		if err != nil {
			glog.Exitf("Failed to read the config file: %v", err)
		}

		glog.Infof("Read probe config: %s", string(b))

		cfg := &configpb.ProbeDef{}
		if err := proto.UnmarshalText(string(b), cfg); err != nil {
			glog.Exit(err)
		}

		_, err = client.AddProbe(context.Background(), &pb.AddProbeRequest{Probe: cfg})
		if err != nil {
			glog.Exit(err)
		}
	}
}
