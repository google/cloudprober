// Copyright 2020 Google Inc.
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
//
// This file implements support for discovering forwarding rules in a GCP
// project. It currently supports only regional forwarding rules. We can
// consider adding support for global forwarding rules in future if necessary.

package gcp

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/gcp/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/filter"
	compute "google.golang.org/api/compute/v1"
)

// frData struct encapsulates information for a fowarding rule.
type frData struct {
	ip     string
	region string
}

/*
ForwardingRulesFilters defines filters supported by the forwarding_rules resource
type.
 Example:
 filter {
	 key: "name"
	 value: "cloudprober.*"
 }
*/
var ForwardingRulesFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	[]string{"name", "region"},
	false,
}

// forwardingRulesLister is a GCE instances lister. It implements a cache,
// that's populated at a regular interval by making the GCE API calls.
// Listing actually only returns the current contents of that cache.
type forwardingRulesLister struct {
	project      string
	c            *configpb.ForwardingRules
	thisInstance string
	l            *logger.Logger

	mu         sync.RWMutex // Mutex for names and cache
	names      []string
	cache      map[string]*frData
	computeSvc *compute.Service
}

// listResources returns the list of resource records, where each record
// consists of an instance name and the IP address associated with it. IP address
// to return is selected based on the provided ipConfig.
func (frl *forwardingRulesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	allFilters, err := filter.ParseFilters(req.GetFilter(), ForwardingRulesFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, regionFilter := allFilters.RegexFilters["name"], allFilters.RegexFilters["region"]

	frl.mu.RLock()
	defer frl.mu.RUnlock()

	for _, name := range frl.names {
		if nameFilter != nil && !nameFilter.Match(name, frl.l) {
			continue
		}

		if regionFilter != nil && !regionFilter.Match(frl.cache[name].region, frl.l) {
			continue
		}

		resources = append(resources, &pb.Resource{
			Name: proto.String(name),
			Ip:   proto.String(frl.cache[name].ip),
		})
	}

	frl.l.Infof("forwarding_rules.listResources: returning %d forwarding rules", len(resources))
	return resources, nil
}

// expand runs equivalent API calls as "gcloud compute instances list",
// and is what is used to populate the cache.
func (frl *forwardingRulesLister) expand(reEvalInterval time.Duration) {
	frl.l.Debugf("forwarding_rules.expand: running for project: %s", frl.project)

	regionList, err := frl.computeSvc.Regions.List(frl.project).Filter(frl.c.GetRegionFilter()).Do()
	if err != nil {
		frl.l.Errorf("forwarding_rules.expand: error while getting list of all regions: %v", err)
		return
	}

	// Temporary cache and names list.
	var (
		names []string
		cache = make(map[string]*frData)
	)

	sleepBetweenRegions := reEvalInterval / (2 * time.Duration(len(regionList.Items)+1))
	for _, region := range regionList.Items {
		frList, err := frl.computeSvc.ForwardingRules.List(frl.project, region.Name).Do()
		if err != nil {
			frl.l.Errorf("forwarding_rules.expand: error while getting list of forwarding rules for region (%s): %v", region.Name, err)
			return
		}
		for _, item := range frList.Items {
			cache[item.Name] = &frData{
				ip:     item.IPAddress,
				region: region.Name,
			}
			names = append(names, item.Name)
		}
		time.Sleep(sleepBetweenRegions)
	}

	// Note that we update the list of names only if after all regions have been
	// expanded successfully. This is to avoid replacing current list with a
	// partial expansion of targets.
	frl.l.Infof("forwarding_rules.expand: got %d forwarding rules", len(names))
	frl.mu.Lock()
	frl.cache = cache
	frl.names = names
	frl.mu.Unlock()
}

func newForwardingRulesLister(project, apiVersion string, c *configpb.ForwardingRules, l *logger.Logger) (*forwardingRulesLister, error) {
	cs, err := defaultComputeService(apiVersion)
	if err != nil {
		return nil, fmt.Errorf("forwarding_rules.expand: error creating compute service: %v", err)
	}

	frl := &forwardingRulesLister{
		project:    project,
		c:          c,
		cache:      make(map[string]*frData),
		computeSvc: cs,
		l:          l,
	}

	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second
	go func() {
		frl.expand(0)
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// forwardingRules, this will make sure that each instance calls GCE
		// API at a different point of time.
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			frl.expand(reEvalInterval)
		}
	}()
	return frl, nil
}
