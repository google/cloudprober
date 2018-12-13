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
Package probes provides an interface to initialize probes using prober config.
*/
package probes

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/dns"
	"github.com/google/cloudprober/probes/external"
	httpprobe "github.com/google/cloudprober/probes/http"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/ping"
	configpb "github.com/google/cloudprober/probes/proto"
	"github.com/google/cloudprober/probes/udp"
	"github.com/google/cloudprober/probes/udplistener"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/targets/lameduck"
	targetspb "github.com/google/cloudprober/targets/proto"
	"github.com/google/cloudprober/validators"
	"github.com/google/cloudprober/web/formatutils"
)

const (
	logsNamePrefix = "cloudprober"
)

var (
	userDefinedProbes   = make(map[string]Probe)
	userDefinedProbesMu sync.Mutex
	extensionMap        = make(map[int]func() Probe)
	extensionMapMu      sync.Mutex
)

func runOnThisHost(runOn string, hostname string) (bool, error) {
	if runOn == "" {
		return true, nil
	}
	r, err := regexp.Compile(runOn)
	if err != nil {
		return false, err
	}
	return r.MatchString(hostname), nil
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

// ProbeInfo encapsulates the probe and associated information.
type ProbeInfo struct {
	Probe
	Name          string
	Type          string
	Interval      string
	Timeout       string
	TargetsDesc   string
	LatencyDistLB string
	LatencyUnit   string
	ProbeConf     string
}

func newLogger(probeName string) (*logger.Logger, error) {
	return logger.New(context.Background(), logsNamePrefix+"."+probeName)
}

func getExtensionProbe(p *configpb.ProbeDef) (Probe, interface{}, error) {
	extensions := proto.RegisteredExtensions(p)
	if len(extensions) > 1 {
		return nil, nil, fmt.Errorf("only one probe extension is allowed per probe, got %d extensions", len(extensions))
	}
	var field int
	var desc *proto.ExtensionDesc
	// There should be only one extension.
	for f, d := range extensions {
		field = int(f)
		desc = d
	}
	if desc == nil {
		return nil, nil, errors.New("no probe extension in probe config")
	}
	value, err := proto.GetExtension(p, desc)
	if err != nil {
		return nil, nil, err
	}
	extensionMapMu.Lock()
	defer extensionMapMu.Unlock()
	newProbeFunc, ok := extensionMap[field]
	if !ok {
		return nil, nil, fmt.Errorf("no probes registered for the extension: %d", field)
	}
	return newProbeFunc(), value, nil
}

// buildProbeOptions builds probe's options using the provided config and some global params.
func buildProbeOptions(p *configpb.ProbeDef, ldLister lameduck.Lister, globalTargetsOpts *targetspb.GlobalTargetsOptions, l *logger.Logger) (*options.Options, error) {
	opts := &options.Options{
		Interval: time.Duration(p.GetIntervalMsec()) * time.Millisecond,
		Timeout:  time.Duration(p.GetTimeoutMsec()) * time.Millisecond,
	}

	var err error
	if opts.Logger, err = newLogger(p.GetName()); err != nil {
		return nil, fmt.Errorf("error in initializing logger for the probe (%s): %v", p.GetName(), err)
	}

	if opts.Targets, err = targets.New(p.GetTargets(), ldLister, globalTargetsOpts, l, opts.Logger); err != nil {
		return nil, err
	}

	if latencyDist := p.GetLatencyDistribution(); latencyDist != nil {
		var d *metrics.Distribution
		if d, err = metrics.NewDistributionFromProto(latencyDist); err != nil {
			return nil, fmt.Errorf("error creating distribution from the specification (%v): %v", latencyDist, err)
		}
		opts.LatencyDist = d
	}

	// latency_unit is specified as a human-readable string, e.g. ns, ms, us etc.
	if opts.LatencyUnit, err = time.ParseDuration("1" + p.GetLatencyUnit()); err != nil {
		return nil, fmt.Errorf("failed to parse the latency unit (%s): %v", p.GetLatencyUnit(), err)
	}

	if len(p.GetValidator()) > 0 {
		opts.Validators, err = validators.Init(p.GetValidator(), opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize validators: %v", err)
		}
	}
	return opts, nil
}

// Init initializes the probes defined in the config.
func Init(probeProtobufs []*configpb.ProbeDef, globalTargetsOpts *targetspb.GlobalTargetsOptions, l *logger.Logger, sysVars map[string]string) (map[string]*ProbeInfo, error) {
	ldLister, err := lameduck.GetDefaultLister()
	if err != nil {
		l.Warningf("Error while getting default lameduck lister, lameduck behavior will be disabled. Err: %v", err)
	}

	probes := make(map[string]*ProbeInfo)

	for _, p := range probeProtobufs {
		// Check if this probe is supposed to run here.
		runHere, err := runOnThisHost(p.GetRunOn(), sysVars["hostname"])
		if err != nil {
			return nil, err
		}
		if !runHere {
			continue
		}

		if probes[p.GetName()] != nil {
			return nil, fmt.Errorf("bad config: probe %s is already defined", p.GetName())
		}

		opts, err := buildProbeOptions(p, ldLister, globalTargetsOpts, l)
		if err != nil {
			return nil, err
		}

		l.Infof("Creating a %s probe: %s", p.GetType(), p.GetName())
		probe, probeConf, err := initProbe(p, opts)
		if err != nil {
			return nil, err
		}

		latencyDistLB := ""
		if opts.LatencyDist != nil {
			latencyDistLB = fmt.Sprintf("%v", opts.LatencyDist.Data().LowerBounds)
		}

		probes[p.GetName()] = &ProbeInfo{
			Probe:         probe,
			Name:          p.GetName(),
			Type:          p.GetType().String(),
			Interval:      opts.Interval.String(),
			Timeout:       opts.Timeout.String(),
			TargetsDesc:   p.Targets.String(),
			LatencyDistLB: latencyDistLB,
			LatencyUnit:   opts.LatencyUnit.String(),
			ProbeConf:     formatutils.ConfToString(probeConf),
		}
	}

	return probes, nil
}

func initProbe(p *configpb.ProbeDef, opts *options.Options) (probe Probe, probeConf interface{}, err error) {
	switch p.GetType() {
	case configpb.ProbeDef_PING:
		probe = &ping.Probe{}
		probeConf = p.GetPingProbe()
	case configpb.ProbeDef_HTTP:
		probe = &httpprobe.Probe{}
		probeConf = p.GetHttpProbe()
	case configpb.ProbeDef_DNS:
		probe = &dns.Probe{}
		probeConf = p.GetDnsProbe()
	case configpb.ProbeDef_EXTERNAL:
		probe = &external.Probe{}
		probeConf = p.GetExternalProbe()
	case configpb.ProbeDef_UDP:
		probe = &udp.Probe{}
		probeConf = p.GetUdpProbe()
	case configpb.ProbeDef_UDP_LISTENER:
		probe = &udplistener.Probe{}
		probeConf = p.GetUdpListenerProbe()
	case configpb.ProbeDef_EXTENSION:
		probe, probeConf, err = getExtensionProbe(p)
		if err != nil {
			return
		}
	case configpb.ProbeDef_USER_DEFINED:
		userDefinedProbesMu.Lock()
		defer userDefinedProbesMu.Unlock()
		probe = userDefinedProbes[p.GetName()]
		if probe == nil {
			err = fmt.Errorf("unregistered user defined probe: %s", p.GetName())
			return
		}
		probeConf = p.GetUserDefinedProbe()
	default:
		err = fmt.Errorf("unknown probe type: %s", p.GetType())
		return
	}

	opts.ProbeConf = probeConf
	err = probe.Init(p.GetName(), opts)
	return
}

// RegisterUserDefined allows you to register a user defined probe with
// cloudprober.
// Example usage:
//	import (
//		"github.com/google/cloudprober"
//		"github.com/google/cloudprober/probes"
//	)
//
//	p := &FancyProbe{}
//	probes.RegisterUserDefined("fancy_probe", p)
//	pr, err := cloudprober.InitFromConfig(*configFile)
//	if err != nil {
//		log.Exitf("Error initializing cloudprober. Err: %v", err)
//	}
func RegisterUserDefined(name string, probe Probe) {
	userDefinedProbesMu.Lock()
	defer userDefinedProbesMu.Unlock()
	userDefinedProbes[name] = probe
}

// RegisterProbeType registers a new probe-type. New probe types are integrated
// with the config subsystem using the protobuf extensions.
//
// TODO(manugarg): Add a full example of using extensions.
func RegisterProbeType(extensionFieldNo int, newProbeFunc func() Probe) {
	extensionMapMu.Lock()
	defer extensionMapMu.Unlock()
	extensionMap[extensionFieldNo] = newProbeFunc
}
