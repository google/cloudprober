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
Binary redis_probe_once implements an example cloudprober external probe that
puts a key-value in a redis server, retrieves it and reports time taken for
both operations.

This probe assumes a redis server running on the localhost listening to the
default port, 6379. To run it, run the following commands in its parent
directory.

# Build external probe
go build ./redis_probe.go

# Launch cloudprober
cloudprober --config_file=cloudprober.cfg

You should see the following output on the stdout (and corresponding prometheus
data on http://localhost:9313/metrics)
cloudprober 1519..0 1519583408 labels=ptype=external,probe=redis_probe,dst= success=1 total=1 latency=12143.765
cloudprober 1519..1 1519583408 labels=ptype=external,probe=redis_probe,dst= set_latency_ms=0.516 get_latency_ms=0.491
cloudprober 1519..2 1519583410 labels=ptype=external,probe=redis_probe,dst= success=2 total=2 latency=30585.915
cloudprober 1519..3 1519583410 labels=ptype=external,probe=redis_probe,dst= set_latency_ms=0.636 get_latency_ms=0.994
cloudprober 1519..4 1519583412 labels=ptype=external,probe=redis_probe,dst= success=3 total=3 latency=42621.871

You can also run this probe in server mode by providing "--server" command line
flag.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	epb "github.com/google/cloudprober/probes/external/proto"
	"github.com/google/cloudprober/probes/external/serverutils"
	"github.com/hoisie/redis"
)

var server = flag.Bool("server", false, "Whether to run in server mode")

func probe() (string, error) {
	var payload []string
	var client redis.Client

	// Measure set latency
	var key = "hello"
	startTime := time.Now()
	err := client.Set(key, []byte("world"))
	if err != nil {
		return "", err
	}

	payload = append(payload, fmt.Sprintf("set_latency_ms %f", float64(time.Since(startTime).Nanoseconds())/1e6))

	// Measure get latency
	startTime = time.Now()
	val, err := client.Get("hello")
	if err != nil {
		return strings.Join(payload, "\n"), err
	}
	payload = append(payload, fmt.Sprintf("get_latency_ms %f", float64(time.Since(startTime).Nanoseconds())/1e6))

	log.Printf("%s=%s", key, string(val))
	return strings.Join(payload, "\n"), nil
}

func main() {
	flag.Parse()
	if *server {
		serverutils.Serve(func(request *epb.ProbeRequest, reply *epb.ProbeReply) {
			payload, err := probe()
			reply.Payload = proto.String(payload)
			if err != nil {
				reply.ErrorMessage = proto.String(err.Error())
			}
		})
	}
	payload, err := probe()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(payload)
}
