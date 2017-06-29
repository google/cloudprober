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

// Package lameduck implements a lameducks provider. Lameduck provider fetches
// lameducks from the RTC (Runtime Configurator) service. This functionality
// allows an operator to do hitless VM upgrades. If a target is set to be in
// lameduck by the operator, it is taken out of the targets list.
package lameduck

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/rtc"
	"google.golang.org/api/runtimeconfig/v1beta1"
)

type lameDucksProvider struct {
	rtc            rtc.Config
	opts           *Options
	expirationTime time.Duration
	l              *logger.Logger

	mu    sync.RWMutex
	names []string
}

// globallameDucksProvider is a global lame ducks provider. It's created only once
// and shared by various GCE targets that require lame duck detection.
var globalLameDucksProvider *lameDucksProvider
var onceLameDucksProvider sync.Once

func (ldp *lameDucksProvider) list() []string {
	ldp.mu.RLock()
	defer ldp.mu.RUnlock()
	return append([]string{}, ldp.names...)
}

// excludeLameDucks finds lame-duck targets by looking at runtimeconfig variables.
func (ldp *lameDucksProvider) expand() {
	resp, err := ldp.rtc.List()
	if err != nil {
		ldp.l.Errorf("targets: Error while getting the runtime config variables for lame-duck targets: %v", err)
		return
	}

	ldp.mu.Lock()
	ldp.names = ldp.processVars(resp)
	ldp.mu.Unlock()
}

func (ldp *lameDucksProvider) processVars(vars []*runtimeconfig.Variable) []string {
	var result []string
	for _, v := range vars {
		ldp.l.Debugf("targets: Processing runtime-config var: %s", v.Name)

		// Variable names include the full path, including the config name.
		varParts := strings.Split(v.Name, "/")
		if len(varParts) == 0 {
			ldp.l.Errorf("targets: Invalid variable name for lame-duck targets: %s", v.Name)
			continue
		}
		ldName := varParts[len(varParts)-1]

		// Variable update time is in RFC3339 format
		// https://cloud.google.com/deployment-manager/runtime-configurator/reference/rest/v1beta1/projects.configs.variables
		updateTime, err := time.Parse(time.RFC3339Nano, v.UpdateTime)
		if err != nil {
			ldp.l.Errorf("targets: Could not parse variable(%s) update time (%s): %v", v.Name, v.UpdateTime, err)
			continue
		}
		if time.Since(updateTime) < time.Duration(ldp.opts.GetExpirationSec())*time.Second {
			ldp.l.Infof("targets: Marking target \"%s\" as lame duck.", ldName)
			result = append(result, ldName)
		} else {
			ldp.l.Infof("targets: Ignoring the stale (%s) lame duck (%s) entry", time.Since(updateTime), ldName)
		}
	}
	return result
}

// List returns the targets that are in lameduck mode.
func List() ([]string, error) {
	if globalLameDucksProvider == nil {
		return nil, errors.New("lameDucksProvider not initialised")
	}
	return globalLameDucksProvider.list(), nil
}

// IsLameducking checks if the target is in lameduck mode.
func IsLameducking(name string) (bool, error) {
	names, err := List()
	if err != nil {
		return false, err
	}
	for _, n := range names {
		if n == name {
			return true, nil
		}
	}
	return false, nil
}

// Lameduck puts the target in lameduck mode.
func Lameduck(name string) error {
	if globalLameDucksProvider == nil {
		return errors.New("lameDucksProvider not initialised")
	}
	return globalLameDucksProvider.rtc.Write(name, []byte{})
}

// UnLameduck removes the target from lameduck mode.
func UnLameduck(name string) error {
	if globalLameDucksProvider == nil {
		return errors.New("lameDucksProvider not initialised")
	}
	err := globalLameDucksProvider.rtc.Delete(name)
	return err
}

// Init initialises the lameduck package.  It has to be called before
// any other function of the package is called.
// TODO: Should instead return an interface, so that this behavior is clear.
func Init(optsProto *Options, l *logger.Logger) error {
	if optsProto == nil {
		return errors.New("lameduck.Init: No lameDuckOptions given")
	}

	proj, err := metadata.ProjectID()
	if err != nil {
		return fmt.Errorf("lameduck.Init: error while getting project id: %v", err)
	}
	cfg := optsProto.GetRuntimeconfigName()

	onceLameDucksProvider.Do(func() {
		rtc, err := rtc.New(proj, cfg)
		if err != nil {
			l.Errorf("targets: unable to build rtc config: %v", err)
			return
		}
		globalLameDucksProvider = &lameDucksProvider{
			rtc:  rtc,
			opts: optsProto,
			l:    l,
		}
		go func() {
			globalLameDucksProvider.expand()
			for _ = range time.Tick(time.Duration(globalLameDucksProvider.opts.GetReEvalSec()) * time.Second) {
				globalLameDucksProvider.expand()
			}
		}()
	})

	return nil
}
