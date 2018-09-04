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

// This binary implements a standalone ResourceDiscovery service (RDS) client.
package main

import (
	"fmt"
	"strings"
	"time"

	"flag"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets/rds/client"
	configpb "github.com/google/cloudprober/targets/rds/client/proto"
	pb "github.com/google/cloudprober/targets/rds/proto"
)

var (
	rdsServer = flag.String("rds_server", "", "gRPC server address")
	provider  = flag.String("provider", "gcp", "Resource provider")
	resType   = flag.String("resource_type", "gce_instances", "Resource type")
	project   = flag.String("project", "", "GCP project")
	filtersF  = flag.String("filters", "", "Comma separated list of filters, e.g. name=ig-us-central1-a-.*")
)

func main() {
	flag.Parse()

	if *project == "" {
		glog.Exit("--project is a required paramater")
	}

	c := &configpb.ClientConf{}

	if *rdsServer != "" {
		c.ServerAddr = rdsServer
	}

	c.Request = &pb.ListResourcesRequest{
		Provider:     proto.String(*provider),
		ResourcePath: proto.String(fmt.Sprintf("%s/%s", *resType, *project)),
	}

	for _, f := range strings.Split(*filtersF, ",") {
		if f == "" {
			continue
		}
		fParts := strings.SplitN(f, "=", 2)
		if len(fParts) != 2 {
			glog.Exitf("bad filter in --filters flag (%s): %s", *filtersF, f)
		}
		c.Request.Filter = append(c.Request.Filter, &pb.Filter{
			Key:   proto.String(fParts[0]),
			Value: proto.String(fParts[1]),
		})
	}

	tgts, err := client.New(c, &logger.Logger{})
	if err != nil {
		glog.Exit(err)
	}
	for {
		for _, name := range tgts.List() {
			ip, _ := tgts.Resolve(name, 4)
			fmt.Printf("%s\t%s\n", name, ip.String())
		}
		time.Sleep(5 * time.Second)
	}
}
