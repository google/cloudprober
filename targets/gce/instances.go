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

package gce

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/targets/gce/proto"
	dnsRes "github.com/google/cloudprober/targets/resolver"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// globalInstancesProvider is an instance of the instancesProvider struct.
// There is only one globalInstancesProvider per project. Like forwardingRules,
// it provides a cache layer that is best shared by all probes in a project.
var globalInstancesProvider = map[string]*instancesProvider{}

// Mutex to safely initialize the globalInstanceProvider
var globalInstancesProviderMu sync.Mutex

// This is how long we wait between API calls per zone.
const defaultAPICallInterval = 250 * time.Microsecond

// instances represents GCE instances. To avoid making GCE API calls for each
// set of GCE instances targets, for example for VM-to-VM probes over internal IP
// and public IP, we use a global instances provider (globalInstancesProvider).
type instances struct {
	projects []string
	pb       *configpb.Instances
	r        *dnsRes.Resolver
}

// newInstances returns a new instances object. It will initialize
// globalInstancesProvider's if needed.
func newInstances(projects []string, opts *configpb.GlobalOptions, ipb *configpb.Instances, globalResolver *dnsRes.Resolver, l *logger.Logger) (*instances, error) {
	reEvalInterval := time.Duration(opts.GetReEvalSec()) * time.Second
	if ipb.GetNetworkInterface() != nil && ipb.GetUseDnsToResolve() {
		return nil, errors.New("network_intf and use_dns_to_resolve are mutually exclusive")
	}
	if ipb.GetUseDnsToResolve() && globalResolver == nil {
		return nil, errors.New("use_dns_to_resolve configured, but globalResolver is nil")
	}
	// Initialize global instances providers if not already initialized.
	if err := initGlobalInstancesProvider(projects, opts.GetApiVersion(), reEvalInterval, l); err != nil {
		return nil, err
	}
	return &instances{
		projects: projects,
		pb:       ipb,
		r:        globalResolver,
	}, nil
}

// List produces a list of all instances. This list is similar to running
// "gcloud compute instances list", but with a cache layer reducing the number
// of actual API calls made.
func (i *instances) List() []string {
	var instancesList []string
	for _, project := range i.projects {
		instancesList = append(instancesList, globalInstancesProvider[project].list()...)
	}
	return instancesList
}

// Resolve resolves the name into an IP address. Unless explicitly configured
// to use DNS, we use the instance object (retrieved through GCE API) to
// determine the instance IPs. If multiple instances in different projects share
// the same name, the instance from the first project mentioned in config will be
// returned
func (i *instances) Resolve(name string, ipVer int) (net.IP, error) {
	if i.pb.GetUseDnsToResolve() {
		return i.r.Resolve(name, ipVer)
	}
	var ins *compute.Instance
	for _, project := range i.projects {
		provider := globalInstancesProvider[project]
		if i := provider.get(name); i != nil {
			ins = i
			break
		}
	}
	if ins == nil {
		return nil, fmt.Errorf("gce.instances.resolve(%s): instance not in in-memory GCE instances database", name)
	}

	// Find the network interface card we are interested in.
	niIndex := 0
	ipType := configpb.Instances_NetworkInterface_PRIVATE
	ni := i.pb.GetNetworkInterface()
	if ni != nil {
		niIndex = int(ni.GetIndex())
		ipType = ni.GetIpType()
	}
	if len(ins.NetworkInterfaces) <= niIndex {
		return nil, fmt.Errorf("gce.instances.resolve(%s): no network interface at index: %d", name, niIndex)
	}
	intf := ins.NetworkInterfaces[niIndex]

	switch ipType {
	case configpb.Instances_NetworkInterface_PRIVATE:
		return net.ParseIP(intf.NetworkIP), nil

	case configpb.Instances_NetworkInterface_PUBLIC:
		if len(intf.AccessConfigs) == 0 {
			return nil, fmt.Errorf("gce.instances.resolve(%s): no access config, instance most likely doesn't have a public IP", name)
		}
		return net.ParseIP(intf.AccessConfigs[0].NatIP), nil

	case configpb.Instances_NetworkInterface_ALIAS:
		if len(intf.AliasIpRanges) == 0 {
			return nil, fmt.Errorf("gce.instances.resolve(%s): no alias IP range", name)
		}
		// Compute API allows specifying CIDR range as an IP address, try that first.
		if ip := net.ParseIP(intf.AliasIpRanges[0].IpCidrRange); ip != nil {
			return ip, nil
		}
		ip, _, err := net.ParseCIDR(intf.AliasIpRanges[0].IpCidrRange)
		return ip, err
	}

	return nil, fmt.Errorf("gce.instances.resolve(%s): unknown IP type for network interface", name)
}

// instancesProvider is a lister which lists GCE instances. There is supposed to
// be only one instancesProvider object per cloudprober instance:
// globalInstancesProvider. It implements a cache, that's populated at a regular
// interval (configured by GlobalGCETargetsOptions.re_eval_sec
// cloudprober/targets/targets.proto) by making GCE API calls. Listing actually
// only returns the current contents of that cache.
type instancesProvider struct {
	project      string
	apiVersion   string
	thisInstance string
	l            *logger.Logger

	mu    sync.RWMutex // Mutex for names and cache
	names []string
	cache map[string]*compute.Instance
}

func initGlobalInstancesProvider(projects []string, apiVersion string, reEvalInterval time.Duration, l *logger.Logger) error {
	globalInstancesProviderMu.Lock()
	defer globalInstancesProviderMu.Unlock()

	var thisInstance string
	if metadata.OnGCE() {
		var err error
		thisInstance, err = metadata.InstanceName()
		if err != nil {
			return fmt.Errorf("initGlobalInstancesProvider: error while getting current instance name: %v", err)
		}
		l.Infof("initGlobalInstancesProvider: this instance: %s", thisInstance)
	}
	for _, project := range projects {
		if globalInstancesProvider[project] != nil {
			continue
		}
		provider := &instancesProvider{
			project:      project,
			apiVersion:   apiVersion,
			thisInstance: thisInstance,
			cache:        make(map[string]*compute.Instance),
			l:            l,
		}
		globalInstancesProvider[project] = provider
		l.Infof("initGlobalInstancesProvider: for project %s", project)
		go func() {
			provider.expand(0)
			// Introduce a random delay between 0-reEvalInterval before
			// starting the refresh loop. If there are multiple cloudprober
			// instances, this will make sure that each instance calls GCE
			// API at a different point of time.
			randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
			time.Sleep(time.Duration(randomDelaySec) * time.Second)
			for _ = range time.Tick(reEvalInterval) {
				provider.expand(reEvalInterval)
			}
		}()
	}
	return nil
}

// get returns compute.Instance resource from the cache by name.
func (ip *instancesProvider) get(name string) *compute.Instance {
	ip.mu.RLock()
	defer ip.mu.RUnlock()
	return ip.cache[name]
}

func (ip *instancesProvider) list() []string {
	ip.mu.RLock()
	defer ip.mu.RUnlock()
	return append([]string{}, ip.names...)
}

// listInstances runs equivalent API calls as "gcloud compute instances list",
// and is what is used to populate the cache.
func listInstances(project, apiVersion string, reEvalInterval time.Duration) ([]*compute.Instance, error) {
	client, err := google.DefaultClient(oauth2.NoContext, compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	cs, err := compute.New(client)
	if err != nil {
		return nil, err
	}
	cs.BasePath = "https://www.googleapis.com/compute/" + apiVersion + "/projects/"
	zonesList, err := cs.Zones.List(project).Do()
	if err != nil {
		return nil, err
	}

	// We wait for this long between zone-specific instance list calls.
	apiCallInterval := defaultAPICallInterval

	// We don't want a longer gap than the following to make sure that all
	// zones finish in one refresh interval.
	maxAPICallInterval := time.Duration(reEvalInterval.Nanoseconds()/int64(len(zonesList.Items))) * time.Nanosecond
	if apiCallInterval > maxAPICallInterval {
		apiCallInterval = maxAPICallInterval
	}

	var result []*compute.Instance
	var instanceList *compute.InstanceList
	for _, zone := range zonesList.Items {
		instanceList, err = cs.Instances.List(project, zone.Name).Filter("status eq \"RUNNING\"").Do()
		if err != nil {
			return nil, err
		}
		result = append(result, instanceList.Items...)
		time.Sleep(apiCallInterval)
	}
	return result, nil
}

// expand will refill the cache, and update names.
func (ip *instancesProvider) expand(reEvalInterval time.Duration) {
	ip.l.Infof("gce.instances.expand[%s]: expanding GCE targets", ip.project)

	computeInstances, err := listInstances(ip.project, ip.apiVersion, reEvalInterval)
	if err != nil {
		ip.l.Errorf("gce.instances.expand[%s]: error while getting list of all instances: %v", ip.project, err)
		return
	}

	var result []string
	ip.mu.Lock()
	defer ip.mu.Unlock()
	for _, ins := range computeInstances {
		if ins.Name == ip.thisInstance {
			continue
		}
		ip.cache[ins.Name] = ins
		result = append(result, ins.Name)
	}

	ip.l.Debugf("Expanded target list: %q", result)
	ip.names = result
}
