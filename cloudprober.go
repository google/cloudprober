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
Package cloudprober provides a prober for running a set of probes.

Cloudprober takes in a config proto which dictates what probes should be created
with what configuration, and manages the asynchronous fan-in/fan-out of the
metrics data from these probes.
*/
package cloudprober

import (
	"context"
	"time"

	"github.com/google/cloudprober/config"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes"
	"github.com/google/cloudprober/servers"
	"github.com/google/cloudprober/surfacers"
	"github.com/google/cloudprober/sysvars"
	"github.com/google/cloudprober/targets/rtc/rtcreporter"
)

const (
	logsNamePrefix    = "cloudprober"
	sysvarsModuleName = "sysvars"
)

// Prober represents a collection of probes where each probe implements the Probe interface.
type Prober struct {
	Probes      map[string]probes.Probe
	c           *config.ProberConfig
	rtcReporter *rtcreporter.Reporter
	surfacers   []surfacers.Surfacer
}

func (pr *Prober) newLogger(probeName string) (*logger.Logger, error) {
	return logger.New(context.Background(), logsNamePrefix+"."+probeName)
}

// InitFromConfig initializes Cloudprober using the provided config.
func InitFromConfig(configFile string) (*Prober, error) {
	pr := &Prober{}
	// Initialize sysvars module
	l, err := pr.newLogger(sysvarsModuleName)
	if err != nil {
		return nil, err
	}
	sysvars.Init(l, nil)

	if pr.c, err = config.Parse(configFile, sysvars.Vars()); err != nil {
		return nil, err
	}

	pr.Probes = probes.Init(pr.c.GetProbe(), pr.c.GetGlobalTargetsOptions(), sysvars.Vars())

	pr.surfacers, err = surfacers.Init(pr.c.GetSurfacer())
	if err != nil {
		return nil, err
	}

	// Initialize RTC reporter, if configured.
	if opts := pr.c.GetRtcReportOptions(); opts != nil {
		l, err := pr.newLogger("rtc-reporter")
		if err != nil {
			return nil, err
		}
		pr.rtcReporter, err = rtcreporter.New(opts, sysvars.Vars(), l)
		if err != nil {
			return nil, err
		}
	}
	return pr, nil
}

// Start starts a previously initialized Cloudprober.
func (pr *Prober) Start(ctx context.Context) {
	dataChan := make(chan *metrics.EventMetrics, 1000)

	go func() {
		var em *metrics.EventMetrics
		for {
			em = <-dataChan
			// Replicate the surfacer message to every surfacer we have
			// registered. Note that s.Write() is expected to be
			// non-blocking to avoid blocking of EventMetrics message
			// processing.
			for _, surfacer := range pr.surfacers {
				surfacer.Write(context.Background(), em)
			}
		}
	}()

	// Start a goroutine to export system variables
	go sysvars.Start(ctx, dataChan, time.Millisecond*time.Duration(pr.c.GetSysvarsIntervalMsec()), pr.c.GetSysvarsEnvVar())

	servers.Start(ctx, pr.c.GetServer(), dataChan)

	// Start RTC reporter if configured.
	if pr.rtcReporter != nil {
		go pr.rtcReporter.Start(ctx)
	}

	// Start probes, each in its own goroutines
	for _, p := range pr.Probes {
		go p.Start(ctx, dataChan)
	}

	// Wait forever
	select {}
}
