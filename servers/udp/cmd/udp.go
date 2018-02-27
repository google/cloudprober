// Copyright 2017 Google Inc.
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
This binary implements a stand-alone UDP server using the
cloudprober/servers/udp/udp package.
*/
package main

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/servers/udp"

	"flag"
	"github.com/golang/glog"
)

var (
	port         = flag.Int("port", 31122, "Port to listen on")
	responseType = flag.String("type", "echo", "Server type: echo|discard")
)

func main() {
	flag.Parse()

	l, err := logger.New(context.Background(), "UDP_"+*responseType)
	if err != nil {
		glog.Fatal(err)
	}

	config := &udp.ServerConf{
		Port: proto.Int32(int32(*port)),
		Type: udp.ServerConf_DISCARD.Enum(),
	}
	server, err := udp.New(context.Background(), config, l)
	if err != nil {
		glog.Fatalf("Error creating a new UDP server: %v", err)
	}
	glog.Fatal(server.Start(context.Background(), nil))
}
