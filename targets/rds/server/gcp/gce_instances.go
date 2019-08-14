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
	"strings"
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

type instanceData struct {
	nis    []*compute.NetworkInterface
	labels map[string]string
}

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
	cache      map[string]*instanceData
	computeSvc *compute.Service
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

func parseFilters(filters []*pb.Filter) (nameFilter *filter.RegexFilter, labelsFilter *filter.LabelsFilter, err error) {
	// TODO(manugarg): Move filtering code to gcp.go once we add more resource
	// types as this logic should be same for most GCP resources.
	labels := make(map[string]string)
	for _, f := range filters {
		if f.GetKey() == "name" {
			nameFilter, err = filter.NewRegexFilter(f.GetValue())
			if err != nil {
				err = fmt.Errorf("gce_instances: error creating regex filter from: %s, err: %v", f.GetValue(), err)
				return
			}
			continue
		}

		// labels.<key> format matches with gcloud's filter options.
		if strings.HasPrefix(f.GetKey(), "labels.") {
			labels[strings.TrimPrefix(f.GetKey(), "labels.")] = f.GetValue()
			continue
		}

		err = fmt.Errorf("unsupported filter key: %s", f.GetKey())
		return
	}

	if len(labels) != 0 {
		labelsFilter, err = filter.NewLabelsFilter(labels)
		if err != nil {
			err = fmt.Errorf("gce_instances: error creating labels filter from: %v, err: %v", labels, err)
			return
		}
	}

	return
}

// listResources returns the list of resource records, where each record
// consists of an instance name and the IP address associated with it. IP address
// to return is selected based on the provided ipConfig.
func (il *gceInstancesLister) listResources(filters []*pb.Filter, ipConfig *pb.IPConfig) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	nameFilter, labelsFilter, err := parseFilters(filters)
	if err != nil {
		return nil, err
	}

	il.mu.RLock()
	defer il.mu.RUnlock()

	for _, name := range il.names {
		if nameFilter != nil && !nameFilter.Match(name, il.l) {
			continue
		}
		if labelsFilter != nil && !labelsFilter.Match(il.cache[name].labels, il.l) {
			continue
		}

		nis := il.cache[name].nis
		ip, err := instanceIP(nis, ipConfig)
		if err != nil {
			return nil, fmt.Errorf("gce_instances (instance %s): error while getting IP - %v", name, err)
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
		cache = make(map[string]*instanceData)
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
			cache[item.Name] = &instanceData{item.NetworkInterfaces, item.Labels}
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
		cache:        make(map[string]*instanceData),
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
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			il.expand(reEvalInterval)
		}
	}()
	return il, nil
}
