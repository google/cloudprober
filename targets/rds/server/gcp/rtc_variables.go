// Copyright 2018 Google Inc.
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

package gcp

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	pb "github.com/google/cloudprober/targets/rds/proto"
	"github.com/google/cloudprober/targets/rds/server/filter"
	configpb "github.com/google/cloudprober/targets/rds/server/gcp/proto"
	"golang.org/x/oauth2/google"
	runtimeconfig "google.golang.org/api/runtimeconfig/v1beta1"
)

// rtcVar is an internal representation of the runtimeconfig (RTC) variables.
// We currently store only the variable name and its update time. In future, we
// may decide to include more information in RTC variables.
type rtcVar struct {
	name       string
	updateTime time.Time
}

// rtcVariablesLister is a RuntimeConfig Variables lister. It implements a
// cache, that's populated at a regular interval by making the GCP API calls.
// Listing actually only returns the current contents of that cache.
type rtcVariablesLister struct {
	project    string
	c          *configpb.RTCVariables
	apiVersion string
	l          *logger.Logger

	mu    sync.RWMutex // Mutex for names and cache
	cache map[string][]*rtcVar
	svc   *runtimeconfig.ProjectsConfigsVariablesService
}

// listResources returns the list of resource records, where each record
// consists of a RTC variable name.
func (rvl *rtcVariablesLister) listResources(filters []*pb.Filter) ([]*pb.Resource, error) {
	var resources []*pb.Resource
	var configNameFilter *filter.RegexFilter
	var freshnessFilter *filter.FreshnessFilter

	for _, f := range filters {
		var err error

		switch f.GetKey() {
		case "config_name":
			configNameFilter, err = filter.NewRegexFilter(f.GetValue())
			if err != nil {
				return nil, fmt.Errorf("rtc_variables: error creating regex filter from: %s, err: %v", f.GetValue(), err)
			}

		case "updated_within":
			freshnessFilter, err = filter.NewFreshnessFilter(f.GetValue())
			if err != nil {
				return nil, fmt.Errorf("rtc_variables: error creating freshness filter from: %s, err: %v", f.GetValue(), err)
			}

		default:
			return nil, fmt.Errorf("rtc_variables: Invalid filter key: %s", f.GetKey())
		}
	}

	rvl.mu.RLock()
	defer rvl.mu.RUnlock()
	for configName, rtcVars := range rvl.cache {
		if configNameFilter != nil && !configNameFilter.Match(configName, rvl.l) {
			continue
		}

		for _, rtcVar := range rtcVars {
			if freshnessFilter != nil && !freshnessFilter.Match(rtcVar.updateTime, rvl.l) {
				continue
			}

			resources = append(resources, &pb.Resource{
				Name: proto.String(rtcVar.name),
			})
		}
	}
	return resources, nil
}

// defaultRTCService returns a compute.Service object, initialized using the
// default credentials.
func defaultRTCService() (*runtimeconfig.ProjectsConfigsVariablesService, error) {
	client, err := google.DefaultClient(context.Background(), runtimeconfig.CloudruntimeconfigScope)
	if err != nil {
		return nil, err
	}
	svc, err := runtimeconfig.New(client)
	if err != nil {
		return nil, err
	}
	return runtimeconfig.NewProjectsConfigsVariablesService(svc), nil
}

// processVar processes the RTC variable resource and returns an rtcVar object.
func processVar(v *runtimeconfig.Variable) (*rtcVar, error) {
	// Variable names include the full path, including the config name.
	varParts := strings.Split(v.Name, "/")
	if len(varParts) == 1 {
		return nil, fmt.Errorf("invalid variable name: %s", v.Name)
	}
	varName := varParts[len(varParts)-1]

	// Variable update time is in RFC3339 format
	// https://cloud.google.com/deployment-manager/runtime-configurator/reference/rest/v1beta1/projects.configs.variables
	updateTime, err := time.Parse(time.RFC3339Nano, v.UpdateTime)
	if err != nil {
		return nil, fmt.Errorf("could not parse variable(%s) update time (%s): %v", v.Name, v.UpdateTime, err)
	}

	return &rtcVar{varName, updateTime}, nil
}

// expand makes API calls to list variables in configured RTC configs and
// populates the cache.
func (rvl *rtcVariablesLister) expand(rtcConfig *configpb.RTCVariables_RTCConfig, reEvalInterval time.Duration) {
	rvl.l.Infof("rtc_variables.expand: expanding RTC vars for project (%s) and config (%s)", rvl.project, rtcConfig.GetName())

	path := "projects/" + rvl.project + "/configs/" + rtcConfig.GetName()
	configVarsList, err := rvl.svc.List(path).Do()
	if err != nil {
		rvl.l.Errorf("rtc_variables.expand: error while getting list of all vars for config path %s: %v", path, err)
		return
	}

	result := make([]*rtcVar, len(configVarsList.Variables))
	for i, v := range configVarsList.Variables {
		rvl.l.Debugf("rtc_variables.expand: processing runtime-config var: %s, update time: %s", v.Name, v.UpdateTime)
		rv, err := processVar(v)
		if err != nil {
			rvl.l.Errorf("Error processing the RTC variable (%s): %v", v.Name, err)
			continue
		}
		result[i] = rv
	}

	rvl.mu.Lock()
	rvl.cache[rtcConfig.GetName()] = result
	rvl.mu.Unlock()
}

func newRTCVariablesLister(project, apiVersion string, c *configpb.RTCVariables, l *logger.Logger) (*rtcVariablesLister, error) {
	svc, err := defaultRTCService()
	if err != nil {
		return nil, fmt.Errorf("rtc_variables.expand: error creating RTC service: %v", err)
	}

	rvl := &rtcVariablesLister{
		project:    project,
		c:          c,
		apiVersion: apiVersion,
		cache:      make(map[string][]*rtcVar),
		svc:        svc,
		l:          l,
	}

	for _, rtcConfig := range rvl.c.GetRtcConfig() {
		reEvalInterval := time.Duration(rtcConfig.GetReEvalSec()) * time.Second
		go func(rtcConfig *configpb.RTCVariables_RTCConfig) {
			rvl.expand(rtcConfig, 0)
			// Introduce a random delay between 0-reEvalInterval before
			// starting the refresh loop. If there are multiple cloudprober
			// rtcVariables, this will make sure that each instance calls GCE
			// API at a different point of time.
			rand.Seed(time.Now().UnixNano())
			randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
			time.Sleep(time.Duration(randomDelaySec) * time.Second)
			for _ = range time.Tick(reEvalInterval) {
				rvl.expand(rtcConfig, reEvalInterval)
			}
		}(rtcConfig)
	}
	return rvl, nil
}
