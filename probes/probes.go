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
Probes package provides an interface to initialize probes using prober config.
*/
package probes

import (
	"context"
	"regexp"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/dns"
	"github.com/google/cloudprober/probes/external"
	httpprobe "github.com/google/cloudprober/probes/http"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/ping"
	"github.com/google/cloudprober/probes/udp"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/targets/lameduck"
)

const (
	logsNamePrefix = "cloudprober"
)

var (
	userDefinedProbes   = make(map[string]Probe)
	userDefinedProbesMu sync.Mutex
)

func runOnThisHost(runOn string, hostname string) bool {
	if runOn == "" {
		return true
	}
	r, err := regexp.Compile(runOn)
	if err != nil {
		glog.Exit(err)
	}
	return r.MatchString(hostname)
}

// Probe interface represents a probe.
//
// A probe is initilized using the Init() method. Init takes the name of the
// probe and probe options.
//
// Start() method starts the probe. Start is not expected to return for the
// lifetime of the prober. It takes a data channel that it writes the probe
// results on. Actual publishing of these results is handled by cloudprober
// itself.
type Probe interface {
	Init(name string, opts *options.Options) error
	Start(ctx context.Context, dataChan chan *metrics.EventMetrics)
}

func newLogger(probeName string) (*logger.Logger, error) {
	return logger.New(context.Background(), logsNamePrefix+"."+probeName)
}

// Init initializes the probes defined in the config.
func Init(probeProtobufs []*ProbeDef, globalTargetsOpts *targets.GlobalTargetsOptions, sysVars map[string]string) map[string]Probe {
	globalTargetsLogger, err := newLogger("globalTargets")
	if err != nil {
		glog.Exitf("Error in initializing logger for the global targets. Err: %v", err)
	}
	if globalTargetsOpts != nil {
		if globalTargetsOpts.GetLameDuckOptions() != nil && metadata.OnGCE() {
			if err := lameduck.Init(globalTargetsOpts.GetLameDuckOptions(), globalTargetsLogger); err != nil {
				glog.Exitf("Error in initializing lameduck module. Err: %v", err)
			}
		}
	}

	probes := make(map[string]Probe)
	for _, p := range probeProtobufs {
		if !runOnThisHost(p.GetRunOn(), sysVars["hostname"]) {
			continue
		}
		if probes[p.GetName()] != nil {
			glog.Exitf("Bad config: probe %s is already defined", p.GetName())
		}
		opts := &options.Options{
			Interval: time.Duration(p.GetIntervalMsec()) * time.Millisecond,
			Timeout:  time.Duration(p.GetTimeoutMsec()) * time.Millisecond,
		}
		var err error
		if opts.Logger, err = newLogger(p.GetName()); err != nil {
			glog.Exitf("Error in initializing logger for the probe %s. Err: %v", p.GetName(), err)
		}

		if opts.Targets, err = targets.New(p.GetTargets(), globalTargetsOpts, globalTargetsLogger, opts.Logger); err != nil {
			glog.Exit(err)
		}
		probes[p.GetName()] = initProbe(p, opts)
	}
	return probes
}

func initProbe(p *ProbeDef, opts *options.Options) (probe Probe) {
	glog.Infof("Creating a %s probe: %s", p.GetType(), p.GetName())

	switch p.GetType() {
	case ProbeDef_PING:
		probe = &ping.Probe{}
		opts.ProbeConf = p.GetPingProbe()
	case ProbeDef_HTTP:
		probe = &httpprobe.Probe{}
		opts.ProbeConf = p.GetHttpProbe()
	case ProbeDef_DNS:
		probe = &dns.Probe{}
		opts.ProbeConf = p.GetDnsProbe()
	case ProbeDef_EXTERNAL:
		probe = &external.Probe{}
		opts.ProbeConf = p.GetExternalProbe()
	case ProbeDef_UDP:
		probe = &udp.Probe{}
		opts.ProbeConf = p.GetUdpProbe()
	case ProbeDef_USER_DEFINED:
		userDefinedProbesMu.Lock()
		defer userDefinedProbesMu.Unlock()
		probe := userDefinedProbes[p.GetName()]
		if probe == nil {
			glog.Exitf("unregistered user defined probe: %s", p.GetName())
		}
		opts.ProbeConf = p.GetUserDefinedProbe()
	default:
		glog.Exitf("Unknown probe type: %s", p.GetType())
	}
	if err := probe.Init(p.GetName(), opts); err != nil {
		glog.Exitf("Error in initializing probe %s from the config. Err: %v", p.GetName(), err)
	}
	return
}

// Register allows you to register a user defined probe with cloudprober.
// Example usage:
//	import (
//		"github.com/google/cloudprober"
//		"github.com/google/cloudprober/probes"
//	)
//
//	p := &FancyProbe{}
//	probes.Register("fancy_probe", p)
//	pr, err := cloudprober.InitFromConfig(*configFile)
//	if err != nil {
//		log.Exitf("Error initializing cloudprober. Err: %v", err)
//	}
func Register(name string, probe Probe) {
	userDefinedProbesMu.Lock()
	defer userDefinedProbesMu.Unlock()
	userDefinedProbes[name] = probe
}
