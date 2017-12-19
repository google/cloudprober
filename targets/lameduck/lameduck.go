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
	"github.com/google/cloudprober/targets/rtc/rtcservice"
	"google.golang.org/api/runtimeconfig/v1beta1"
)

// Service is an interface for lameducking operations on VMs.
type Service interface {
	Lameduck(name string) error
	Unlameduck(name string) error
	IsLameducking(name string) (bool, error)
	List() ([]string, error)
}

// A singletion like lameduck Service.
var globalService Service
var globalServiceMu sync.RWMutex

// Package level functions.

// Lameduck puts the target in lameduck mode.
func Lameduck(name string) error {
	ldSvc, err := getGlobalService()
	if err != nil {
		return err
	}
	return ldSvc.Lameduck(name)
}

// Unlameduck removes the target from lameduck mode.
func Unlameduck(name string) error {
	ldSvc, err := getGlobalService()
	if err != nil {
		return err
	}
	return ldSvc.Unlameduck(name)
}

// IsLameducking checks if the target is in lameduck mode.
func IsLameducking(name string) (bool, error) {
	ldSvc, err := getGlobalService()
	if err != nil {
		return false, err
	}
	return ldSvc.IsLameducking(name)
}

// List returns the targets that are in lameduck mode.
func List() ([]string, error) {
	ldSvc, err := getGlobalService()
	if err != nil {
		return nil, err
	}
	return ldSvc.List()
}

// Implements the Service interface.
type service struct {
	rtc            rtcservice.Config
	opts           *Options
	expirationTime time.Duration
	l              *logger.Logger

	mu    sync.RWMutex
	names []string
}

// Updates the list of lameduck targets' names.
func (ldSvc *service) expand() {
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
func (ldSvc *service) processVars(vars []*runtimeconfig.Variable) []string {
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
func (ldSvc *service) Lameduck(name string) error {
	return ldSvc.rtc.Write(name, []byte{})
}

// Unlameduck removes the target from lameduck mode.
func (ldSvc *service) Unlameduck(name string) error {
	err := ldSvc.rtc.Delete(name)
	return err
}

// IsLameducking checks if the target is in lameduck mode.
func (ldSvc *service) IsLameducking(name string) (bool, error) {
	names, err := ldSvc.List()
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

// List returns the targets that are in lameduck mode.
func (ldSvc *service) List() ([]string, error) {
	ldSvc.mu.RLock()
	defer ldSvc.mu.RUnlock()
	return append([]string{}, ldSvc.names...), nil
}

// NewService returns a new lameduck Service interface.
func NewService(optsProto *Options, l *logger.Logger) (Service, error) {
	if optsProto == nil {
		return nil, fmt.Errorf("lameduck.Init: failed to construct lameduck Service: no lameDuckOptions given")
	}

	var proj string
	if optsProto.GetRuntimeconfigProject() == "" {
		var err error
		proj, err = metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("lameduck.Init: error while getting project id: %v", err)
		}
	}
	cfg := optsProto.GetRuntimeconfigName()

	rtc, err := rtcservice.New(proj, cfg)
	if err != nil {
		return nil, fmt.Errorf("lameduck.Init : rtcconfig service initialization failed : %v", err)
	}

	ldSvc := &service{
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

// Init initializes the lameduck package.
func Init(optsProto *Options, l *logger.Logger) error {
	globalServiceMu.Lock()
	defer globalServiceMu.Unlock()
	// Make sure we only initialize globalService once.
	if globalService != nil {
		return nil
	}

	ldSvc, err := NewService(optsProto, l)
	if err != nil {
		return err
	}
	globalService = ldSvc
	return nil
}

func getGlobalService() (Service, error) {
	globalServiceMu.RLock()
	defer globalServiceMu.RUnlock()
	// Check if globvalService was initialized.
	if globalService == nil {
		return nil, errors.New("global lameduck service wasn't initialized")
	}
	return globalService, nil
}

// SetGlobalService overrides the global lameduck Service interface.
func SetGlobalService(ldSvc Service) {
	globalServiceMu.Lock()
	defer globalServiceMu.Unlock()
	globalService = ldSvc
}
