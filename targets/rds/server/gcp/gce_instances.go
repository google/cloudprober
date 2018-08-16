// Copyright 2017-2018 Google Inc.
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
	pb "github.com/google/cloudprober/targets/rds/proto"
	"github.com/google/cloudprober/targets/rds/server/filter"
	configpb "github.com/google/cloudprober/targets/rds/server/gcp/proto"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// This is how long we wait between API calls per zone.
const defaultAPICallInterval = 250 * time.Microsecond

// gceInstancesLister is a GCE instances lister. It implements a cache,
// that's populated at a regular interval by making the GCE API calls.
// Listing actually only returns the current contents of that cache.
type gceInstancesLister struct {
	project      string
	c            *configpb.GCEInstances
	apiVersion   string
	thisInstance string
	l            *logger.Logger

	mu         sync.RWMutex // Mutex for names and cache
	names      []string
	cache      map[string][]*compute.NetworkInterface
	computeSvc *compute.Service
}

// listResources returns the list of resource records, where each record
// consists of an instance name and the IP address associated with it. IP address
// to return is selected based on the provided ipConfig.
func (il *gceInstancesLister) listResources(filters []*pb.Filter, ipConfig *pb.IPConfig) ([]*pb.Resource, error) {
	var resources []*pb.Resource
	var nameFilter *filter.RegexFilter

	for _, f := range filters {
		switch f.GetKey() {
		case "name":
			var err error
			nameFilter, err = filter.NewRegexFilter(f.GetValue())
			if err != nil {
				return nil, fmt.Errorf("gce_instances: error creating regex filter from: %s, err: %v", f.GetValue(), err)
			}

		default:
			return nil, fmt.Errorf("gce_instances: Invalid filter key: %s", f.GetKey())
		}
	}

	niIndex := 0
	ipType := pb.IPConfig_DEFAULT
	if ipConfig != nil {
		niIndex = int(ipConfig.GetNicIndex())
		ipType = ipConfig.GetIpType()
	}

	il.mu.RLock()
	defer il.mu.RUnlock()

	for _, name := range il.names {
		if nameFilter != nil && !nameFilter.Match(name, il.l) {
			continue
		}

		nis := il.cache[name]
		if len(nis) <= niIndex {
			return nil, fmt.Errorf("gce_instances: instance %s doesn't have network interface at index %d", name, niIndex)
		}
		ni := nis[niIndex]

		var ip string
		switch ipType {
		case pb.IPConfig_DEFAULT:
			ip = ni.NetworkIP
		case pb.IPConfig_PUBLIC:
			if len(ni.AccessConfigs) == 0 {
				return nil, fmt.Errorf("gce_instances (instance: %s, network_interface: %d): no public IP", name, niIndex)
			}
			ip = ni.AccessConfigs[0].NatIP
		case pb.IPConfig_ALIAS:
			if len(ni.AliasIpRanges) == 0 {
				return nil, fmt.Errorf("gce_instances: instance %s has no alias IP range", name)
			}
			// Compute API allows specifying CIDR range as an IP address, try that first.
			if cidrIP := net.ParseIP(ni.AliasIpRanges[0].IpCidrRange); cidrIP != nil {
				ip = cidrIP.String()
				continue
			}
			ciderIP, _, err := net.ParseCIDR(ni.AliasIpRanges[0].IpCidrRange)
			if err != nil {
				return nil, fmt.Errorf("gce_instances (instance: %s, network_interface: %d): error geting alias IP: %v", name, niIndex, err)
			}
			ip = ciderIP.String()
		}

		resources = append(resources, &pb.Resource{
			Name: proto.String(name),
			Ip:   proto.String(ip),
			// TODO(manugarg): Add support for returning instance id as well. I want to
			// implement feature parity with the current targets first and then add
			// more features.
		})
	}
	return resources, nil
}

// defaultComputeService returns a compute.Service object, initialized using
// default credentials.
func defaultComputeService() (*compute.Service, error) {
	client, err := google.DefaultClient(context.Background(), compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	cs, err := compute.New(client)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

// expand runs equivalent API calls as "gcloud compute instances list",
// and is what is used to populate the cache.
func (il *gceInstancesLister) expand(reEvalInterval time.Duration) {
	il.l.Infof("gce_instances.expand: expanding GCE targets for project: %s", il.project)

	zonesList, err := il.computeSvc.Zones.List(il.project).Filter(il.c.GetZoneFilter()).Do()
	if err != nil {
		il.l.Errorf("gce_instances.expand: error while getting list of all zones: %v", err)
		return
	}

	// Temporary cache and names list.
	var (
		names []string
		cache = make(map[string][]*compute.NetworkInterface)
	)
	sleepBetweenZones := reEvalInterval / (2 * time.Duration(len(zonesList.Items)+1))
	for _, zone := range zonesList.Items {
		instanceList, err := il.computeSvc.Instances.List(il.project, zone.Name).Filter("status eq \"RUNNING\"").Do()
		if err != nil {
			il.l.Errorf("gce_instances.expand: error while getting list of all instances: %v", err)
			return
		}
		for _, item := range instanceList.Items {
			if item.Name == il.thisInstance {
				continue
			}
			cache[item.Name] = item.NetworkInterfaces
			names = append(names, item.Name)
		}
		time.Sleep(sleepBetweenZones)
	}

	// Note that we update the list of names only if after all zones have been
	// expanded successfully. This is to avoid replacing current list with a
	// partial expansion of targets. This is in contrast with instance-toNetInf
	// cache, which is updated as we go through the instance list.
	il.l.Debugf("Expanded target list: %q", names)
	il.mu.Lock()
	il.cache = cache
	il.names = names
	il.mu.Unlock()
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

	cs, err := defaultComputeService()
	if err != nil {
		return nil, fmt.Errorf("gce_instances.expand: error creating compute service: %v", err)
	}

	il := &gceInstancesLister{
		project:      project,
		c:            c,
		apiVersion:   apiVersion,
		thisInstance: thisInstance,
		cache:        make(map[string][]*compute.NetworkInterface),
		computeSvc:   cs,
		l:            l,
	}

	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second
	go func() {
		il.expand(0)
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// gceInstances, this will make sure that each instance calls GCE
		// API at a different point of time.
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for _ = range time.Tick(reEvalInterval) {
			il.expand(reEvalInterval)
		}
	}()
	return il, nil
}
