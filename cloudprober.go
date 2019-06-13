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
	"sync"

	"github.com/google/cloudprober/prober"
	"github.com/google/cloudprober/probes"
	"github.com/google/cloudprober/servers"
	"github.com/google/cloudprober/surfacers"
)

const (
	sysvarsModuleName = "sysvars"
)

// Global prober.Prober instance protected by a mutex.
var cloudProber struct {
	*prober.Prober
	sync.Mutex
}

// InitFromConfig initializes Cloudprober using the provided config.
func InitFromConfig(configFile string) error {
	// Return immediately if prober is already initialized.
	cloudProber.Lock()
	defer cloudProber.Unlock()

	if cloudProber.Prober != nil {
		return nil
	}

	pr := &prober.Prober{}
	if err := pr.Init(configFile); err != nil {
		return err
	}
	cloudProber.Prober = pr
	return nil
}

// Start starts a previously initialized Cloudprober.
func Start(ctx context.Context) {
	cloudProber.Lock()
	defer cloudProber.Unlock()
	if cloudProber.Prober == nil {
		panic("Prober is not initialized. Did you call cloudprober.InitFromConfig first?")
	}
	cloudProber.Start(ctx)
}

// GetConfig returns the prober config.
func GetConfig() string {
	cloudProber.Lock()
	defer cloudProber.Unlock()
	return cloudProber.TextConfig
}

// GetInfo returns information on all the probes, servers and surfacers.
func GetInfo() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {
	cloudProber.Lock()
	defer cloudProber.Unlock()
	return cloudProber.Probes, cloudProber.Surfacers, cloudProber.Servers
}
