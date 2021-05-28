// Copyright 2017-2020 The Cloudprober Authors.
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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	neturl "net/url"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/gcp/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/filter"
	"golang.org/x/oauth2/google"
)

// This is how long we wait between API calls per zone.
const defaultAPICallInterval = 250 * time.Microsecond
const computeScope = "https://www.googleapis.com/auth/compute.readonly"

type accessConfig struct {
	NatIP        string
	ExternalIpv6 string `json:"externalIpv6,omitempty"`
}

type networkInterface struct {
	NetworkIP   string `json:"networkIP,omitempty"`
	Ipv6Address string `json:"ipv6Address,omitempty"`

	AliasIPRanges []struct {
		IPCidrRange string `json:"ipCidrRange,omitempty"`
	} `json:"aliasIpRanges,omitempty"`

	AccessConfigs     []accessConfig
	Ipv6AccessConfigs []accessConfig `json:"ipv6AccessConfigs,omitempty"`
}

// instanceInfo represents instance items that we fetch from the API.
type instanceInfo struct {
	Name              string
	Labels            map[string]string
	NetworkInterfaces []networkInterface
}

// instanceData represents objects that we store in cache.
type instanceData struct {
	ii          *instanceInfo
	lastUpdated int64
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
	baseAPIPath  string
	httpClient   *http.Client
	getURLFunc   func(client *http.Client, url string) ([]byte, error)
	l            *logger.Logger

	mu            sync.RWMutex
	namesPerScope map[string][]string                 // "us-e1-b": ["i1", i2"]
	cachePerScope map[string]map[string]*instanceData // "us-e1-b": {"i1: data}
}

// ipV picks an IP address from an array of v4 and v6 addresses, based on the
// asked IP version.
// Note: we should consider moving this to a common location.
func ipV(ips [2]string, ipVer pb.IPConfig_IPVersion) string {
	switch ipVer {
	case pb.IPConfig_IPV4:
		return ips[0]
	case pb.IPConfig_IPV6:
		return ips[1]
	default:
		if ips[0] != "" {
			return ips[0]
		}
		return ips[1]
	}
}

func externalAddr(nic networkInterface, ipVer pb.IPConfig_IPVersion) (string, error) {
	ips := [2]string{"null", "null"}
	if len(nic.AccessConfigs) != 0 {
		ips[0] = nic.AccessConfigs[0].NatIP
	}
	if len(nic.Ipv6AccessConfigs) != 0 {
		ips[1] = nic.Ipv6AccessConfigs[0].ExternalIpv6
	}
	ip := ipV(ips, ipVer)
	if ip == "null" {
		return "", fmt.Errorf("no %s public IP", ipVer.String())
	}
	return ip, nil
}

func instanceIP(nis []networkInterface, ipConfig *pb.IPConfig) (string, error) {
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
		return ipV([2]string{ni.NetworkIP, ni.Ipv6Address}, ipConfig.GetIpVersion()), nil

	case pb.IPConfig_PUBLIC:
		return externalAddr(ni, ipConfig.GetIpVersion())

	case pb.IPConfig_ALIAS:
		if len(ni.AliasIPRanges) == 0 {
			return "", fmt.Errorf("no alias IP for NIC(%d)", niIndex)
		}
		// Compute API allows specifying CIDR range as an IP address, try that first.
		if cidrIP := net.ParseIP(ni.AliasIPRanges[0].IPCidrRange); cidrIP != nil {
			return cidrIP.String(), nil
		}

		cidrIP, _, err := net.ParseCIDR(ni.AliasIPRanges[0].IPCidrRange)
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
			ins := cache[name].ii
			if ins == nil {
				il.l.Errorf("gce_instances: cached info missing for %s", name)
				continue
			}

			if nameFilter != nil && !nameFilter.Match(name, il.l) {
				continue
			}
			if labelsFilter != nil && !labelsFilter.Match(ins.Labels, il.l) {
				continue
			}

			nis := ins.NetworkInterfaces
			ip, err := instanceIP(nis, req.GetIpConfig())
			if err != nil {
				return nil, fmt.Errorf("gce_instances (instance %s): error while getting IP - %v", name, err)
			}

			resources = append(resources, &pb.Resource{
				Name:        proto.String(name),
				Ip:          proto.String(ip),
				Labels:      ins.Labels,
				LastUpdated: proto.Int64(cache[name].lastUpdated),
				// TODO(manugarg): Add support for returning instance id as well. I want to
				// implement feature parity with the current targets first and then add
				// more features.
			})
		}
	}

	il.l.Infof("gce_instances.listResources: returning %d instances", len(resources))
	return resources, nil
}

func parseZonesJSON(resp []byte) ([]string, error) {
	var itemList struct {
		Items []struct {
			Name string
		}
	}

	if err := json.Unmarshal(resp, &itemList); err != nil {
		return nil, fmt.Errorf("error while parsing zones list result: %v", err)
	}

	keys := make([]string, len(itemList.Items))
	for i, item := range itemList.Items {
		keys[i] = item.Name
	}

	return keys, nil
}

func parseInstancesJSON(resp []byte) (keys []string, instances map[string]*instanceInfo, err error) {
	var itemList struct {
		Items []*instanceInfo
	}

	if err = json.Unmarshal(resp, &itemList); err != nil {
		return
	}

	keys = make([]string, len(itemList.Items))
	instances = make(map[string]*instanceInfo)
	for i, item := range itemList.Items {
		keys[i] = item.Name
		instances[keys[i]] = item
	}

	return
}

func getURLWithClient(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error while fetching URL %s, status: %s", url, resp.Status)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	return respBytes, nil
}

func (il *gceInstancesLister) expandForZone(zone string) ([]string, map[string]*instanceData, error) {
	var (
		names []string
		cache = make(map[string]*instanceData)
	)

	url := fmt.Sprintf("%s/zones/%s/instances?filter=%s", il.baseAPIPath, zone, neturl.PathEscape("status eq \"RUNNING\""))
	respBytes, err := il.getURLFunc(il.httpClient, url)
	if err != nil {
		return nil, nil, err
	}

	keys, instances, err := parseInstancesJSON(respBytes)
	if err != nil {
		return nil, nil, err
	}

	ts := time.Now().Unix()
	for _, name := range keys {
		if name == il.thisInstance {
			continue
		}
		cache[name] = &instanceData{instances[name], ts}
		names = append(names, name)
	}

	return names, cache, nil
}

// expand runs equivalent API calls as "gcloud compute instances list",
// and is what is used to populate the cache.
func (il *gceInstancesLister) expand(reEvalInterval time.Duration) {
	il.l.Infof("gce_instances.expand: running for the project: %s", il.project)

	url := il.baseAPIPath + "/zones"
	if il.c.GetZoneFilter() != "" {
		url = fmt.Sprintf("%s?filter=%s", url, neturl.PathEscape(il.c.GetZoneFilter()))
	}

	respBytes, err := il.getURLFunc(il.httpClient, url)
	if err != nil {
		il.l.Errorf("gce_instances.expand: error while listing zones: %v", err)
		return
	}

	zones, err := parseZonesJSON(respBytes)
	if err != nil {
		il.l.Errorf("gce_instances.expand: error while parsing zones list response: %v", err)
		return
	}

	// Shuffle the zones list to change the order in each cycle.
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })

	il.l.Infof("gce_instances.expand: expanding GCE targets for %d zones", len(zones))

	var numItems int

	sleepBetweenZones := reEvalInterval / (2 * time.Duration(len(zones)+1))

	for _, zone := range zones {
		names, cache, err := il.expandForZone(zone)
		if err != nil {
			il.l.Errorf("gce_instances.expand: error while listing instances in zone %s: %v", zone, err)
			continue
		}

		il.mu.Lock()
		il.namesPerScope[zone] = names
		il.cachePerScope[zone] = cache
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

	client, err := google.DefaultClient(context.Background(), computeScope)
	if err != nil {
		return nil, fmt.Errorf("error creating default HTTP OAuth client: %v", err)
	}

	il := &gceInstancesLister{
		project:       project,
		c:             c,
		thisInstance:  thisInstance,
		baseAPIPath:   "https://www.googleapis.com/compute/" + apiVersion + "/projects/" + project,
		httpClient:    client,
		getURLFunc:    getURLWithClient,
		cachePerScope: make(map[string]map[string]*instanceData),
		namesPerScope: make(map[string][]string),
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
