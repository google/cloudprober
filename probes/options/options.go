// Copyright 2017-2019 Google Inc.
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
Package options provides a shared interface to common probe options.
*/
package options

import (
	"fmt"
	"net"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/probeutils"
	configpb "github.com/google/cloudprober/probes/proto"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/targets/lameduck"
	targetspb "github.com/google/cloudprober/targets/proto"
	"github.com/google/cloudprober/validators"
)

// Options encapsulates common probe options.
type Options struct {
	Targets           targets.Targets
	Interval, Timeout time.Duration
	Logger            *logger.Logger
	ProbeConf         interface{} // Probe-type specific config
	LatencyDist       *metrics.Distribution
	LatencyUnit       time.Duration
	Validators        []*validators.ValidatorWithName
	SourceIP          net.IP
}

// getSourceFromConfig returns the source IP from the config either directly
// or by resolving the network interface to an IP, depending on which is provided.
func getSourceIPFromConfig(p *configpb.ProbeDef, l *logger.Logger) (net.IP, error) {
	switch p.SourceIpConfig.(type) {

	case *configpb.ProbeDef_SourceIp:
		sourceIP := net.ParseIP(p.GetSourceIp())
		if sourceIP == nil {
			return nil, fmt.Errorf("invalid source IP: %s", p.GetSourceIp())
		}
		return sourceIP, nil

	case *configpb.ProbeDef_SourceInterface:
		return probeutils.ResolveIntfAddr(p.GetSourceInterface())

	default:
		return nil, fmt.Errorf("unknown source type: %v", p.GetSourceIpConfig())
	}
}

// BuildProbeOptions builds probe's options using the provided config and some global params.
func BuildProbeOptions(p *configpb.ProbeDef, ldLister lameduck.Lister, globalTargetsOpts *targetspb.GlobalTargetsOptions, l *logger.Logger) (*Options, error) {
	opts := &Options{
		Interval: time.Duration(p.GetIntervalMsec()) * time.Millisecond,
		Timeout:  time.Duration(p.GetTimeoutMsec()) * time.Millisecond,
	}

	var err error
	if opts.Logger, err = logger.NewCloudproberLog(p.GetName()); err != nil {
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

	if p.GetSourceIpConfig() != nil {
		opts.SourceIP, err = getSourceIPFromConfig(p, l)
		if err != nil {
			return nil, fmt.Errorf("failed to get source address for the probe: %v", err)
		}
	}

	return opts, nil
}
