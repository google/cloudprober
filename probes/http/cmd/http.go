// Copyright 2017-2018 Google Inc.
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

// This program implements a stand-alone http prober binary using the
// cloudprober/http package. It's intended to help prototype the http package
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
	"github.com/google/cloudprober/probes/http"
	configpb "github.com/google/cloudprober/probes/http/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets"
)

var (
	config    = flag.String("config_file", "", "Config file (ProbeConf)")
	intervalF = flag.Duration("interval", time.Second*2, "Interval between probes")
	timeoutF  = flag.Duration("timeout", time.Second*1, "Per-probe timeout")
	targetsF  = flag.String("targets", "www.google.com", "Static host targets.")
)

func main() {
	flag.Parse()

	c := &configpb.ProbeConf{}

	// If we are given a config file, read it. If not, use defaults.
	if *config != "" {
		b, err := ioutil.ReadFile(*config)
		if err != nil {
			glog.Exit(err)
		}
		if err = proto.UnmarshalText(string(b), c); err != nil {
			glog.Exitf("Error while parsing config protobuf %s: Err: %v", string(b), err)
		}
	}

	opts := &options.Options{
		Interval:  *intervalF,
		Timeout:   *timeoutF,
		Targets:   targets.StaticTargets(*targetsF),
		ProbeConf: c,
	}

	hp := &http.Probe{}
	if err := hp.Init("http_test", opts); err != nil {
		glog.Exitf("Error in initializing probe %s from the config. Err: %v", "http_test", err)
	}
	dataChan := make(chan *metrics.EventMetrics, 1000)
	go hp.Start(context.Background(), dataChan)

	for {
		em := <-dataChan
		fmt.Println(em.String())
	}
}
