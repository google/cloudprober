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
Package prober provides a prober for running a set of probes.

Prober takes in a config proto which dictates what probes should be created
with what configuration, and manages the asynchronous fan-in/fan-out of the
metrics data from these probes.
*/
package prober

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"sync"
	"time"

	"github.com/golang/glog"
	configpb "github.com/google/cloudprober/config/proto"
	"github.com/google/cloudprober/config/runconfig"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	spb "github.com/google/cloudprober/prober/proto"
	"github.com/google/cloudprober/probes"
	"github.com/google/cloudprober/probes/options"
	probes_configpb "github.com/google/cloudprober/probes/proto"
	"github.com/google/cloudprober/servers"
	"github.com/google/cloudprober/surfacers"
	"github.com/google/cloudprober/sysvars"
	"github.com/google/cloudprober/targets/lameduck"
	rdsserver "github.com/google/cloudprober/targets/rds/server"
	"github.com/google/cloudprober/targets/rtc/rtcreporter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Prober represents a collection of probes where each probe implements the Probe interface.
type Prober struct {
	Probes      map[string]*probes.ProbeInfo
	Servers     []*servers.ServerInfo
	c           *configpb.ProberConfig
	l           *logger.Logger
	mu          sync.Mutex
	rdsServer   *rdsserver.Server
	ldLister    lameduck.Lister
	rtcReporter *rtcreporter.Reporter
	Surfacers   []*surfacers.SurfacerInfo

	// Probe channel to handle starting of the new probes.
	grpcStartProbeCh chan string

	// Per-probe cancelFunc map.
	probeCancelFunc map[string]context.CancelFunc

	// dataChan for passing metrics between probes and main goroutine.
	dataChan chan *metrics.EventMetrics

	// Used by GetConfig for /config handler.
	TextConfig string
}

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

func (pr *Prober) addProbe(p *probes_configpb.ProbeDef) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// Check if this probe is supposed to run here.
	runHere, err := runOnThisHost(p.GetRunOn(), sysvars.Vars()["hostname"])
	if err != nil {
		return err
	}
	if !runHere {
		return nil
	}

	if pr.Probes[p.GetName()] != nil {
		return status.Errorf(codes.AlreadyExists, "probe %s is already defined", p.GetName())
	}

	opts, err := options.BuildProbeOptions(p, pr.ldLister, pr.c.GetGlobalTargetsOptions(), pr.l)
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}

	pr.l.Infof("Creating a %s probe: %s", p.GetType(), p.GetName())
	probeInfo, err := probes.CreateProbe(p, opts)
	if err != nil {
		return status.Errorf(codes.Unknown, err.Error())
	}
	pr.Probes[p.GetName()] = probeInfo

	return nil
}

// Init initialize prober with the given config file.
func (pr *Prober) Init(ctx context.Context, cfg *configpb.ProberConfig, l *logger.Logger) error {
	pr.c = cfg
	pr.l = l

	// Initialize lameduck lister
	globalTargetsOpts := pr.c.GetGlobalTargetsOptions()

	if globalTargetsOpts.GetLameDuckOptions() != nil {
		ldLogger, err := logger.NewCloudproberLog("lame-duck")
		if err != nil {
			return fmt.Errorf("error in initializing lame-duck logger: %v", err)
		}

		if err := lameduck.InitDefaultLister(globalTargetsOpts.GetLameDuckOptions(), nil, ldLogger); err != nil {
			return err
		}

		pr.ldLister, err = lameduck.GetDefaultLister()
		if err != nil {
			pr.l.Warningf("Error while getting default lameduck lister, lameduck behavior will be disabled. Err: %v", err)
		}
	}

	var err error

	// Initiliaze probes
	pr.Probes = make(map[string]*probes.ProbeInfo)
	pr.probeCancelFunc = make(map[string]context.CancelFunc)
	for _, p := range pr.c.GetProbe() {
		if err := pr.addProbe(p); err != nil {
			return err
		}
	}

	// Initialize servers
	pr.Servers, err = servers.Init(ctx, pr.c.GetServer())
	if err != nil {
		return err
	}

	pr.Surfacers, err = surfacers.Init(pr.c.GetSurfacer())
	if err != nil {
		return err
	}

	// Initialize cloudprober gRPC service if configured.
	srv := runconfig.DefaultGRPCServer()
	if srv != nil {
		pr.grpcStartProbeCh = make(chan string)
		spb.RegisterCloudproberServer(srv, pr)
	}

	// Initialize RDS server, if configured.
	if c := pr.c.GetRdsServer(); c != nil {
		l, err := logger.NewCloudproberLog("rds-server")
		if err != nil {
			return err
		}
		if pr.rdsServer, err = rdsserver.New(ctx, c, nil, l); err != nil {
			return err
		}
	}

	// Initialize RTC reporter, if configured.
	if opts := pr.c.GetRtcReportOptions(); opts != nil {
		l, err := logger.NewCloudproberLog("rtc-reporter")
		if err != nil {
			return err
		}
		if pr.rtcReporter, err = rtcreporter.New(opts, sysvars.Vars(), l); err != nil {
			return err
		}
	}
	return nil
}

// Start starts a previously initialized Cloudprober.
func (pr *Prober) Start(ctx context.Context) {
	pr.dataChan = make(chan *metrics.EventMetrics, 1000)

	go func() {
		var em *metrics.EventMetrics
		for {
			em = <-pr.dataChan
			var s = em.String()
			if len(s) > logger.MaxLogEntrySize {
				glog.Warningf("Metric entry for timestamp %v dropped due to large size: %d", em.Timestamp, len(s))
				continue
			}

			// Replicate the surfacer message to every surfacer we have
			// registered. Note that s.Write() is expected to be
			// non-blocking to avoid blocking of EventMetrics message
			// processing.
			for _, surfacer := range pr.Surfacers {
				surfacer.Write(context.Background(), em)
			}
		}
	}()

	// Start a goroutine to export system variables
	go sysvars.Start(ctx, pr.dataChan, time.Millisecond*time.Duration(pr.c.GetSysvarsIntervalMsec()), pr.c.GetSysvarsEnvVar())

	// Start servers, each in its own goroutine
	for _, s := range pr.Servers {
		go s.Start(ctx, pr.dataChan)
	}

	// Start RDS server if configured.
	if pr.rdsServer != nil {
		go pr.rdsServer.Start(ctx, pr.dataChan)
	}

	// Start RTC reporter if configured.
	if pr.rtcReporter != nil {
		go pr.rtcReporter.Start(ctx)
	}

	if pr.c.GetDisableJitter() {
		for name := range pr.Probes {
			go pr.startProbe(ctx, name)
		}
		return
	}
	pr.startProbesWithJitter(ctx)

	if runconfig.DefaultGRPCServer() != nil {
		// Start a goroutine to handle starting of the probes added through gRPC.
		// AddProbe adds new probes to the pr.grpcStartProbeCh channel and this
		// goroutine reads from that channel and starts the probe using the overall
		// Start context.
		go func() {
			for {
				select {
				case name := <-pr.grpcStartProbeCh:
					pr.startProbe(ctx, name)
				}
			}
		}()
	}
}

func (pr *Prober) startProbe(ctx context.Context, name string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	probeCtx, cancelFunc := context.WithCancel(ctx)
	pr.probeCancelFunc[name] = cancelFunc
	go pr.Probes[name].Start(probeCtx, pr.dataChan)
}

// startProbesWithJitter try to space out probes over time, as much as possible,
// without making it too complicated. We arrange probes into interval buckets -
// all probes with the same interval will be part of the same bucket, and we
// then spread out probes within that interval by introducing a delay of
// interval / len(probes) between probes. We also introduce a random jitter
// between different interval buckets.
func (pr *Prober) startProbesWithJitter(ctx context.Context) {
	// Seed random number generator.
	rand.Seed(time.Now().UnixNano())

	// Make interval -> [probe1, probe2, probe3..] map
	intervalBuckets := make(map[time.Duration][]*probes.ProbeInfo)
	for _, p := range pr.Probes {
		intervalBuckets[p.Options.Interval] = append(intervalBuckets[p.Options.Interval], p)
	}

	for interval, probeInfos := range intervalBuckets {
		go func(interval time.Duration, probeInfos []*probes.ProbeInfo) {
			// Introduce a random jitter between interval buckets.
			randomDelayMsec := rand.Int63n(int64(interval.Seconds() * 1000))
			time.Sleep(time.Duration(randomDelayMsec) * time.Millisecond)

			interProbeDelay := interval / time.Duration(len(probeInfos))

			// Spread out probes evenly with an interval bucket.
			for _, p := range probeInfos {
				pr.l.Info("Starting probe: ", p.Name)
				go pr.startProbe(ctx, p.Name)
				time.Sleep(interProbeDelay)
			}
		}(interval, probeInfos)
	}
}
