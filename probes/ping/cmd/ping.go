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

// This program implements a stand-alone ping prober binary using the
// cloudprober/ping package. It's intended to help prototype the ping package
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
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/ping"
	configpb "github.com/google/cloudprober/probes/ping/proto"
	"github.com/google/cloudprober/targets"
)

var (
	targetsF = flag.String("targets", "www.google.com,www.yahoo.com", "Comma separated list of targets")
	config   = flag.String("config_file", "", "Config file to change ping probe options (see probes/ping/config.proto for default options)")
)

func main() {
	flag.Parse()

	probeConfig := &configpb.ProbeConf{}
	if *config != "" {
		b, err := ioutil.ReadFile(*config)
		if err != nil {
			glog.Exit(err)
		}
		if err = proto.UnmarshalText(string(b), probeConfig); err != nil {
			glog.Exitf("error while parsing config: %v", err)
		}
	}

	opts := &options.Options{
		Targets:     targets.StaticTargets(*targetsF),
		Interval:    2 * time.Second,
		Timeout:     time.Second,
		LatencyUnit: 1 * time.Millisecond,
		ProbeConf:   probeConfig,
	}
	p := &ping.Probe{}
	if err := p.Init("ping", opts); err != nil {
		glog.Exitf("error initializing ping probe from config: %v", err)
	}

	dataChan := make(chan *metrics.EventMetrics, 1000)
	go p.Start(context.Background(), dataChan)

	for {
		em := <-dataChan
		fmt.Println(em.String())
	}
}
