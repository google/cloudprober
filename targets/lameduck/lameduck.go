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
	"net/http"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets/rtc/rtcservice"
	runtimeconfig "google.golang.org/api/runtimeconfig/v1beta1"
)

// Lister is an interface for getting current lameducks.
type Lister interface {
	List() ([]string, error)
}

// global.lister is a singleton Lister. It caches data from the upstream config
// service, allowing for multiple consumers to lookup for lameducks without
// increasing load on the upstream service.
var global struct {
	mu     sync.RWMutex
	lister Lister
}

// Service provides methods to do lameduck operations on VMs.
type Service struct {
	rtc            rtcservice.Config
	opts           *Options
	expirationTime time.Duration
	l              *logger.Logger

	mu    sync.RWMutex
	names []string
}

// Updates the list of lameduck targets' names.
func (ldSvc *Service) expand() {
	resp, err := ldSvc.rtc.List()
	if err != nil {
		ldSvc.l.Errorf("targets: Error while getting the runtime config variables for lame-duck targets: %v", err)
		return
	}
	ldSvc.mu.Lock()
	ldSvc.names = ldSvc.processVars(resp)
	ldSvc.mu.Unlock()
}

// Returns the list of un-expired names of lameduck targets.
func (ldSvc *Service) processVars(vars []*runtimeconfig.Variable) []string {
	var result []string
	for _, v := range vars {
		ldSvc.l.Debugf("targets: Processing runtime-config var: %s", v.Name)

		// Variable names include the full path, including the config name.
		varParts := strings.Split(v.Name, "/")
		if len(varParts) == 0 {
			ldSvc.l.Errorf("targets: Invalid variable name for lame-duck targets: %s", v.Name)
			continue
		}
		ldName := varParts[len(varParts)-1]

		// Variable update time is in RFC3339 format
		// https://cloud.google.com/deployment-manager/runtime-configurator/reference/rest/v1beta1/projects.configs.variables
		updateTime, err := time.Parse(time.RFC3339Nano, v.UpdateTime)
		if err != nil {
			ldSvc.l.Errorf("targets: Could not parse variable(%s) update time (%s): %v", v.Name, v.UpdateTime, err)
			continue
		}
		if time.Since(updateTime) < time.Duration(ldSvc.opts.GetExpirationSec())*time.Second {
			ldSvc.l.Infof("targets: Marking target \"%s\" as lame duck.", ldName)
			result = append(result, ldName)
		} else {
			ldSvc.l.Infof("targets: Ignoring the stale (%s) lame duck (%s) entry", time.Since(updateTime), ldName)
		}
	}
	return result
}

// Lameduck puts the target in lameduck mode.
func (ldSvc *Service) Lameduck(name string) error {
	return ldSvc.rtc.Write(name, []byte{0})
}

// Unlameduck removes the target from lameduck mode.
func (ldSvc *Service) Unlameduck(name string) error {
	err := ldSvc.rtc.Delete(name)
	return err
}

// List returns the targets that are in lameduck mode.
func (ldSvc *Service) List() ([]string, error) {
	ldSvc.mu.RLock()
	defer ldSvc.mu.RUnlock()
	return append([]string{}, ldSvc.names...), nil
}

// NewService creates a new lameduck Service using the provided config options
// and an oauth2 enabled *http.Client; if the client is set to nil, an oauth
// enabled client is created automatically using GCP default credentials.
func NewService(optsProto *Options, c *http.Client, l *logger.Logger) (*Service, error) {
	if optsProto == nil {
		return nil, fmt.Errorf("lameduck.Init: failed to construct lameduck Service: no lameDuckOptions given")
	}

	proj := optsProto.GetRuntimeconfigProject()
	if proj == "" {
		var err error
		proj, err = metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("lameduck.Init: error while getting project id: %v", err)
		}
	}
	cfg := optsProto.GetRuntimeconfigName()

	rtc, err := rtcservice.New(proj, cfg, c)
	if err != nil {
		return nil, fmt.Errorf("lameduck.Init : rtcconfig service initialization failed : %v", err)
	}

	ldSvc := &Service{
		rtc:  rtc,
		opts: optsProto,
		l:    l,
	}
	ldSvc.expand()

	// Update the lameduck targets every [opts.ReEvalSec] seconds.
	go func() {
		for _ = range time.Tick(time.Duration(ldSvc.opts.GetReEvalSec()) * time.Second) {
			ldSvc.expand()
		}
	}()
	return ldSvc, nil
}

// InitDefaultLister initializes the package using the given arguments. If a
// lister is given in the arguments, global.lister is set to that, otherwise a
// new lameduck service is created using the config options, and global.lister
// is set to that service. Initiating the package from a given lister is useful
// for testing pacakges that depend on this package.
func InitDefaultLister(optsProto *Options, lister Lister, l *logger.Logger) error {
	global.mu.Lock()
	defer global.mu.Unlock()
	// Make sure we only initialize global.lister once.
	if global.lister != nil {
		return nil
	}

	if lister != nil {
		global.lister = lister
		return nil
	}

	ldSvc, err := NewService(optsProto, nil, l)
	if err != nil {
		return err
	}
	global.lister = ldSvc
	return nil
}

// GetDefaultLister returns the global Lister. If global lister is
// uninitialized, it returns an error.
func GetDefaultLister() (Lister, error) {
	global.mu.RLock()
	defer global.mu.RUnlock()
	if global.lister == nil {
		return nil, errors.New("global lameduck service not initialized")
	}
	return global.lister, nil
}
