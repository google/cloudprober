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
Binary cloudprober is a tool for running a set of probes and metric surfacers
on a GCE VM. Cloudprober takes in a config proto which dictates what probes
and surfacers should be created with what configuration, and then manages the
asynchronous fan-in/fan-out of the data between the probes and the surfacers.
*/
package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/pprof"

	"flag"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
	"github.com/google/cloudprober"
	"github.com/google/cloudprober/common/file"
	"github.com/google/cloudprober/config"
	"github.com/google/cloudprober/config/runconfig"
	"github.com/google/cloudprober/sysvars"
	"github.com/google/cloudprober/web"
)

var (
	configFile       = flag.String("config_file", "", "Config file")
	versionFlag      = flag.Bool("version", false, "Print version and exit")
	cpuprofile       = flag.String("cpuprof", "", "Write cpu profile to file")
	memprofile       = flag.String("memprof", "", "Write heap profile to file")
	configTest       = flag.Bool("configtest", false, "Dry run to test config file")
	dumpConfig       = flag.Bool("dumpconfig", false, "Dump processed config to stdout")
	testInstanceName = flag.String("test_instance_name", "ig-us-central1-a-01-0000", "Instance name example to be used in tests")
	nonCloud         = flag.Bool("non_cloud", false, "Disable cloud metadata collection")

	// configTestVars provides a sane set of sysvars for config testing.
	configTestVars = map[string]string(nil)
)

// This gets overwritten by using -ldflags="-X main.version=${VERSION}" at
// the build time.
var version = "undefined"

func setupConfigTestVars() {
	configTestVars = map[string]string{
		"zone":              "us-central1-a",
		"project":           "fake-domain.com:fake-project",
		"project_id":        "12345678",
		"instance":          *testInstanceName,
		"internal_ip":       "192.168.0.10",
		"external_ip":       "10.10.10.10",
		"instance_template": "ig-us-central1-a-01",
	}
}

const (
	configMetadataKeyName = "cloudprober_config"
	defaultConfigFile     = "/etc/cloudprober.cfg"
)

func setupProfiling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	var f *os.File
	if *cpuprofile != "" {
		var err error
		f, err = os.Create(*cpuprofile)
		if err != nil {
			glog.Exit(err)
		}
		if err = pprof.StartCPUProfile(f); err != nil {
			glog.Errorf("Could not start CPU profiling: %v", err)
		}
	}
	go func(file *os.File) {
		<-sigChan
		pprof.StopCPUProfile()
		if *cpuprofile != "" {
			if err := file.Close(); err != nil {
				glog.Exit(err)
			}
		}
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				glog.Exit(err)
			}
			if err = pprof.WriteHeapProfile(f); err != nil {
				glog.Exit(err)
			}
			if err := f.Close(); err != nil {
				glog.Exit(err)
			}
		}
		os.Exit(1)
	}(f)
}

func configFileToString(fileName string) string {
	b, err := file.ReadFile(fileName)
	if err != nil {
		glog.Exitf("Failed to read the config file: %v", err)
	}
	return string(b)
}

func getConfig() string {
	if *configFile != "" {
		return configFileToString(*configFile)
	}
	// On GCE first check if there is a config in custom metadata
	// attributes.
	if metadata.OnGCE() {
		if config, err := config.ReadFromGCEMetadata(configMetadataKeyName); err != nil {
			glog.Infof("Error reading config from metadata. Err: %v", err)
		} else {
			return config
		}
	}
	// If config not found in metadata, check default config on disk
	if _, err := os.Stat(defaultConfigFile); !os.IsNotExist(err) {
		return configFileToString(defaultConfigFile)
	}
	glog.Warningf("Config file %s not found. Using default config.", defaultConfigFile)
	return config.DefaultConfig()
}

func main() {
	flag.Parse()

	runconfig.SetVersion(version)
	runconfig.SetNonCloud(*nonCloud)

	if *versionFlag {
		fmt.Println(version)
		return
	}

	setupConfigTestVars()

	if *dumpConfig {
		sysvars.Init(nil, configTestVars)
		text, err := config.ParseTemplate(getConfig(), sysvars.Vars())
		if err != nil {
			glog.Exitf("Error parsing config file. Err: %v", err)
		}
		fmt.Println(text)
		return
	}

	if *configTest {
		sysvars.Init(nil, configTestVars)
		_, err := config.ParseForTest(configFileToString(*configFile), sysvars.Vars())
		if err != nil {
			glog.Exitf("Error parsing config file. Err: %v", err)
		}
		return
	}

	setupProfiling()

	err := cloudprober.InitFromConfig(getConfig())
	if err != nil {
		glog.Exitf("Error initializing cloudprober. Err: %v", err)
	}

	// web.Init sets up web UI for cloudprober.
	web.Init()

	cloudprober.Start(context.Background())

	// Wait forever
	select {}
}
