// Copyright 2017-2020 Google Inc.
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
	"net"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/gcp/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/filter"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// This is how long we wait between API calls per zone.
const defaultAPICallInterval = 250 * time.Microsecond

type instanceData struct {
	nis    []*compute.NetworkInterface
	labels map[string]string
}

/*
GCEInstancesFilters defines filters supported by the gce_instances resource
type.
 Example:
 filter {
	 key: "name"
	 value: "cloudprober.*"
 }
 filter {
	 key: "labels.app"
	 value: "service-a"
 }
*/
var GCEInstancesFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	[]string{"name"},
	true,
}

// gceInstancesLister is a GCE instances lister. It implements a cache,
// that's populated at a regular interval by making the GCE API calls.
// Listing actually only returns the current contents of that cache.
type gceInstancesLister struct {
	project      string
	c            *configpb.GCEInstances
	thisInstance string
	l            *logger.Logger

	mu            sync.RWMutex
	namesPerScope map[string][]string                 // "us-e1-b": ["i1", i2"]
	cachePerScope map[string]map[string]*instanceData // "us-e1-b": {"i1: data}
	computeSvc    *compute.Service
}

func instanceIP(nis []*compute.NetworkInterface, ipConfig *pb.IPConfig) (string, error) {
	var niIndex int
	ipType := pb.IPConfig_DEFAULT
	if ipConfig != nil {
		niIndex = int(ipConfig.GetNicIndex())
		ipType = ipConfig.GetIpType()
	}

	if len(nis) <= niIndex {
		return "", fmt.Errorf("no network interface at index %d", niIndex)
	}

	ni := nis[niIndex]

	switch ipType {
	case pb.IPConfig_DEFAULT:
		return ni.NetworkIP, nil

	case pb.IPConfig_PUBLIC:
		if len(ni.AccessConfigs) == 0 {
			return "", fmt.Errorf("no public IP for NIC(%d)", niIndex)
		}
		return ni.AccessConfigs[0].NatIP, nil

	case pb.IPConfig_ALIAS:
		if len(ni.AliasIpRanges) == 0 {
			return "", fmt.Errorf("no alias IP for NIC(%d)", niIndex)
		}
		// Compute API allows specifying CIDR range as an IP address, try that first.
		if cidrIP := net.ParseIP(ni.AliasIpRanges[0].IpCidrRange); cidrIP != nil {
			return cidrIP.String(), nil
		}

		cidrIP, _, err := net.ParseCIDR(ni.AliasIpRanges[0].IpCidrRange)
		if err != nil {
			return "", fmt.Errorf("error geting alias IP for NIC(%d): %v", niIndex, err)
		}
		return cidrIP.String(), nil
	}

	return "", nil
}

// listResources returns the list of resource records, where each record
// consists of an instance name and the IP address associated with it. IP address
// to return is selected based on the provided ipConfig.
func (il *gceInstancesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	allFilters, err := filter.ParseFilters(req.GetFilter(), GCEInstancesFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.LabelsFilter

	il.mu.RLock()
	defer il.mu.RUnlock()

	for zone, names := range il.namesPerScope {
		cache := il.cachePerScope[zone]

		for _, name := range names {
			ins := cache[name]
			if ins == nil {
				il.l.Errorf("gce_instances: cached info missing for %s", name)
				continue
			}

			if nameFilter != nil && !nameFilter.Match(name, il.l) {
				continue
			}
			if labelsFilter != nil && !labelsFilter.Match(ins.labels, il.l) {
				continue
			}

			nis := ins.nis
			ip, err := instanceIP(nis, req.GetIpConfig())
			if err != nil {
				return nil, fmt.Errorf("gce_instances (instance %s): error while getting IP - %v", name, err)
			}

			resources = append(resources, &pb.Resource{
				Name:   proto.String(name),
				Ip:     proto.String(ip),
				Labels: ins.labels,
				// TODO(manugarg): Add support for returning instance id as well. I want to
				// implement feature parity with the current targets first and then add
				// more features.
			})
		}
	}

	il.l.Infof("gce_instances.listResources: returning %d instances", len(resources))
	return resources, nil
}

// defaultComputeService returns a compute.Service object, initialized using
// default credentials.
func defaultComputeService(apiVersion string) (*compute.Service, error) {
	client, err := google.DefaultClient(context.Background(), compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	cs, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	cs.BasePath = "https://www.googleapis.com/compute/" + apiVersion + "/projects/"
	return cs, nil
}

func (il *gceInstancesLister) expandForZone(zone string) ([]string, map[string]*instanceData, error) {
	var (
		names []string
		cache = make(map[string]*instanceData)
	)

	instanceList, err := il.computeSvc.Instances.List(il.project, zone).
		Filter("status eq \"RUNNING\"").Do()
	if err != nil {
		return nil, nil, err
	}
	for _, item := range instanceList.Items {
		if item.Name == il.thisInstance {
			continue
		}
		cache[item.Name] = &instanceData{item.NetworkInterfaces, item.Labels}
		names = append(names, item.Name)
	}

	return names, cache, nil
}

// expand runs equivalent API calls as "gcloud compute instances list",
// and is what is used to populate the cache.
func (il *gceInstancesLister) expand(reEvalInterval time.Duration) {
	il.l.Infof("gce_instances.expand: running for the project: %s", il.project)

	zonesList, err := il.computeSvc.Zones.List(il.project).Filter(il.c.GetZoneFilter()).Do()
	if err != nil {
		il.l.Errorf("gce_instances.expand: error while getting list of all zones: %v", err)
		return
	}

	// Shuffle the zones list to change the order in each cycle.
	zl := zonesList.Items
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(zl), func(i, j int) { zl[i], zl[j] = zl[j], zl[i] })

	il.l.Infof("gce_instances.expand: expanding GCE targets for %d zones", len(zl))

	var numItems int

	sleepBetweenZones := reEvalInterval / (2 * time.Duration(len(zl)+1))

	for _, zone := range zl {
		names, cache, err := il.expandForZone(zone.Name)
		if err != nil {
			il.l.Errorf("gce_instances.expand: error while listing instances in zone %s: %v", zone.Name, err)
			continue
		}

		il.mu.Lock()
		il.namesPerScope[zone.Name] = names
		il.cachePerScope[zone.Name] = cache
		il.mu.Unlock()

		numItems += len(names)
		time.Sleep(sleepBetweenZones)
	}

	il.l.Infof("gce_instances.expand: got %d instances", numItems)
}

func newGCEInstancesLister(project, apiVersion string, c *configpb.GCEInstances, l *logger.Logger) (*gceInstancesLister, error) {
	var thisInstance string
	if metadata.OnGCE() {
		var err error
		thisInstance, err = metadata.InstanceName()
		if err != nil {
			return nil, fmt.Errorf("newGCEInstancesLister: error while getting current instance name: %v", err)
		}
		l.Infof("newGCEInstancesLister: this instance: %s", thisInstance)
	}

	cs, err := defaultComputeService(apiVersion)
	if err != nil {
		return nil, fmt.Errorf("gce_instances.expand: error creating compute service: %v", err)
	}

	il := &gceInstancesLister{
		project:       project,
		c:             c,
		thisInstance:  thisInstance,
		cachePerScope: make(map[string]map[string]*instanceData),
		namesPerScope: make(map[string][]string),
		computeSvc:    cs,
		l:             l,
	}

	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second
	go func() {
		il.expand(0)
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// gceInstances, this will make sure that each instance calls GCE
		// API at a different point of time.
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			il.expand(reEvalInterval)
		}
	}()
	return il, nil
}
