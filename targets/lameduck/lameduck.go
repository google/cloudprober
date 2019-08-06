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
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/targets/lameduck/proto"
	rdsclient "github.com/google/cloudprober/targets/rds/client"
	rdsclient_configpb "github.com/google/cloudprober/targets/rds/client/proto"
	rdspb "github.com/google/cloudprober/targets/rds/proto"
	"github.com/google/cloudprober/targets/rtc/rtcservice"
	runtimeconfig "google.golang.org/api/runtimeconfig/v1beta1"
)

// Lister is an interface for getting current lameducks.
type Lister interface {
	List() []string
}

// Lameducker provides an interface to Lameduck/Unlameduck an instance.
//
// Cloudprober doesn't currently (as of July, 2018) use this interface by
// itself. It's provided here so that other software (e.g. probing deployment
// management software) can lameduck/unlameduck instances in a way that
// Cloudprober understands.
type Lameducker interface {
	Lameduck(name string) error
	Unlameduck(name string) error
}

// global.lister is a singleton Lister. It caches data from the upstream config
// service, allowing for multiple consumers to lookup for lameducks without
// increasing load on the upstream service.
var global struct {
	mu     sync.RWMutex
	lister Lister
}

// service provides methods to do lameduck operations on VMs.
type service struct {
	rtc            rtcservice.Config
	opts           *configpb.Options
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
	expirationTime := time.Duration(ldSvc.opts.GetExpirationSec()) * time.Second
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
		if time.Since(updateTime) < expirationTime {
			ldSvc.l.Infof("targets: Marking target \"%s\" as lame duck.", ldName)
			result = append(result, ldName)
			continue
		}
		// Log only if variable is not older than 10 times of expiration time. This
		// is to avoid keep logging old expired entries.
		if time.Since(updateTime) < 10*expirationTime {
			ldSvc.l.Infof("targets: Ignoring the stale (%s) lame duck (%s) entry", time.Since(updateTime), ldName)
		}
	}
	return result
}

// Lameduck puts the target in lameduck mode.
func (ldSvc *service) Lameduck(name string) error {
	return ldSvc.rtc.Write(name, []byte{0})
}

// Unlameduck removes the target from lameduck mode.
func (ldSvc *service) Unlameduck(name string) error {
	err := ldSvc.rtc.Delete(name)
	return err
}

// List returns the targets that are in lameduck mode.
func (ldSvc *service) List() []string {
	ldSvc.mu.RLock()
	defer ldSvc.mu.RUnlock()
	return append([]string{}, ldSvc.names...)
}

// NewService creates a new lameduck service using the provided config options
// and an oauth2 enabled *http.Client; if the client is set to nil, an oauth
// enabled client is created automatically using GCP default credentials.
func newService(opts *configpb.Options, proj string, hc *http.Client, l *logger.Logger) (*service, error) {
	if opts == nil {
		return nil, fmt.Errorf("lameduck.Init: failed to construct lameduck Service: no lameDuckOptions given")
	}
	if l == nil {
		l = &logger.Logger{}
	}

	cfg := opts.GetRuntimeconfigName()

	rtc, err := rtcservice.New(proj, cfg, hc)
	if err != nil {
		return nil, fmt.Errorf("lameduck.Init : rtcconfig service initialization failed : %v", err)
	}

	return &service{
		rtc:  rtc,
		opts: opts,
		l:    l,
	}, nil
}

func getProject(opts *configpb.Options) (string, error) {
	project := opts.GetRuntimeconfigProject()
	if project == "" {
		var err error
		project, err = metadata.ProjectID()
		if err != nil {
			return "", fmt.Errorf("lameduck.getProject: error while getting project id: %v", err)
		}
	}
	return project, nil
}

// NewLameducker creates a new lameducker using the provided config and an
// oauth2 enabled *http.Client; if the client is set to nil, an oauth enabled
// client is created automatically using GCP default credentials.
func NewLameducker(opts *configpb.Options, hc *http.Client, l *logger.Logger) (Lameducker, error) {
	project, err := getProject(opts)
	if err != nil {
		return nil, err
	}
	return newService(opts, project, hc, l)
}

func rdsClient(opts *configpb.Options, project string, l *logger.Logger) (*rdsclient.Client, error) {
	rdsClientConf := &rdsclient_configpb.ClientConf{
		ServerAddr: proto.String(opts.GetRdsServerAddr()),
		Request: &rdspb.ListResourcesRequest{
			Provider:     proto.String("gcp"),
			ResourcePath: proto.String(fmt.Sprintf("rtc_variables/%s", project)),
			Filter: []*rdspb.Filter{
				{
					Key:   proto.String("config_name"),
					Value: proto.String(opts.GetRuntimeconfigName()),
				},
				{
					Key:   proto.String("updated_within"),
					Value: proto.String(fmt.Sprintf("%ds", opts.GetExpirationSec())),
				},
			},
		},
		ReEvalSec: proto.Int32(opts.GetReEvalSec()),
	}

	return rdsclient.New(rdsClientConf, nil, l)
}

// InitDefaultLister initializes the package using the given arguments. If a
// lister is given in the arguments, global.lister is set to that, otherwise a
// new lameduck service is created using the config options, and global.lister
// is set to that service. Initiating the package from a given lister is useful
// for testing pacakges that depend on this package.
func InitDefaultLister(opts *configpb.Options, lister Lister, l *logger.Logger) error {
	global.mu.Lock()
	defer global.mu.Unlock()

	// Make sure we initialize global.lister only once.
	if global.lister != nil {
		return nil
	}

	// If a lister has been provided, use that. It's useful for testing.
	if lister != nil {
		global.lister = lister
		return nil
	}

	project, err := getProject(opts)
	if err != nil {
		return err
	}

	if opts.GetUseRds() {
		c, err := rdsClient(opts, project, l)
		if err != nil {
			return err
		}
		global.lister = c
		return nil
	}

	// Create a new lister and set it up for auto-refresh.
	ldSvc, err := newService(opts, project, nil, l)
	if err != nil {
		return err
	}

	ldSvc.expand()
	go func() {
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// instances, this will make sure that each instance calls GCE
		// API at a different point of time.
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(ldSvc.opts.GetReEvalSec()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)

		// Update the lameducks every [opts.ReEvalSec] seconds.
		for range time.Tick(time.Duration(ldSvc.opts.GetReEvalSec()) * time.Second) {
			ldSvc.expand()
		}
	}()

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
