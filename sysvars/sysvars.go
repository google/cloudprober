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

// Package sysvars implements a system variables exporter. It exports variables defined
// through an environment variable, as well other system variables like process uptime.
package sysvars

import (
	"context"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

var (
	sysVarsMu sync.RWMutex
	sysVars   map[string]string
	l         *logger.Logger
	startTime time.Time
)

// Vars returns a copy of the system variables map, if already initialized.
// Otherwise an empty map is returned.
func Vars() map[string]string {
	vars := make(map[string]string)
	// We should never have to wait for these locks as sysVars are
	// updated inside Init and Init should be called only once, in
	// the beginning.
	sysVarsMu.RLock()
	defer sysVarsMu.RUnlock()
	if sysVars == nil {
		// Log an error and return an empty map if sysVars is not initialized yet.
		l.Error("Sysvars map is un-initialized. sysvars.Vars() was called before sysvars.Init().")
		return vars
	}
	for k, v := range sysVars {
		vars[k] = v
	}
	return vars
}

func parseEnvVars(envVarsName string) map[string]string {
	envVars := make(map[string]string)
	if os.Getenv(envVarsName) == "" {
		return envVars
	}
	l.Infof("%s: %s", envVarsName, os.Getenv(envVarsName))
	for _, v := range strings.Split(os.Getenv(envVarsName), ",") {
		kv := strings.Split(v, "=")
		if len(kv) != 2 {
			l.Warningf("Bad env var: %s, skipping", v)
			continue
		}
		envVars[kv[0]] = kv[1]
	}
	return envVars
}

// Init initializes the sysvars module's global data structure. Init makes sure
// to initialize only once, further calls are a no-op. If needed, userVars
// can be passed to Init to add custom variables to sysVars. This can be useful
// for tests which require sysvars that might not exist, or might have the wrong
// value.
func Init(ll *logger.Logger, userVars map[string]string) error {
	sysVarsMu.Lock()
	defer sysVarsMu.Unlock()
	if sysVars != nil {
		return nil
	}

	startTime = time.Now()
	l = ll
	sysVars = make(map[string]string)
	if err := SystemVars(sysVars); err != nil {
		l.Error(err)
		return err
	}

	for k, v := range userVars {
		sysVars[k] = v
	}

	return nil
}

// Run exports system variables at the given interval. It overlays variables with
// variables passed through the envVarsName env variable.
func Run(ctx context.Context, dataChan chan *metrics.EventMetrics, interval time.Duration, envVarsName string) {
	vars := Vars()
	for k, v := range parseEnvVars(envVarsName) {
		vars[k] = v
	}
	var varsKeys []string
	for k := range vars {
		varsKeys = append(varsKeys, k)
	}
	sort.Strings(varsKeys)

	for ts := range time.Tick(interval) {
		// Don't run another probe if context is canceled already.
		select {
		case <-ctx.Done():
			return
		default:
		}

		em := metrics.NewEventMetrics(ts).
			AddLabel("ptype", "sysvars").
			AddLabel("probe", "sysvars")
		em.Kind = metrics.GAUGE
		for _, k := range varsKeys {
			em.AddMetric(k, metrics.NewString(vars[k]))
		}

		// Uptime is since this module started.
		timeSince := time.Since(startTime).Seconds()
		em.AddMetric("uptime", metrics.NewInt(int64(timeSince)))

		dataChan <- em
		l.Info(em.String())
	}
}
