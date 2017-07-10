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

// Dns_bin implements a stand-alone dns prober binary using the
// cloudprober/probes/dns package. It's intended to help prototype the dns package
// quickly and doesn't provide the facilities that cloudprober provides.
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"flag"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/dns"
	"github.com/google/cloudprober/targets"
)

var (
	config    = flag.String("config_file", "", "Config file (ProbeConf)")
	intervalF = flag.Duration("interval", 2*time.Second, "Interval between probes")
	timeoutF  = flag.Duration("timeout", 1*time.Second, "Per-probe timeout")
	targetsF  = flag.String("targets", "8.8.8.8", "Static host targets.")
)

func main() {
	flag.Parse()

	c := &dns.ProbeConf{}

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

	interval := *intervalF
	timeout := *timeoutF
	tgts := targets.StaticTargets(*targetsF)

	dp := &dns.Probe{}
	if err := dp.Init("dns_test", tgts, interval, timeout, nil, c); err != nil {
		glog.Exitf("Error in initializing probe %s from the config. Err: %v", "dns_test", err)
	}
	dataChan := make(chan *metrics.EventMetrics, 1000)
	go dp.Start(context.Background(), dataChan)

	for {
		em := <-dataChan
		fmt.Println(em.String())
	}
}
