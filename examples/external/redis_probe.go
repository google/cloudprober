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
*/
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hoisie/redis"
)

func main() {
	var client redis.Client
	var key = "hello"
	startTime := time.Now()
	client.Set(key, []byte("world"))
	fmt.Printf("set_latency_ms %f\n", float64(time.Since(startTime).Nanoseconds())/1e6)

	startTime = time.Now()
	val, _ := client.Get("hello")
	log.Printf("%s=%s", key, string(val))
	fmt.Printf("get_latency_ms %f\n", float64(time.Since(startTime).Nanoseconds())/1e6)
}
