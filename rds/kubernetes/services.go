// Copyright 2019 Google Inc.
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

package kubernetes

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/kubernetes/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/filter"
)

type servicesLister struct {
	c         *configpb.Services
	namespace string
	kClient   *client

	mu    sync.RWMutex // Mutex for names and cache
	keys  []resourceKey
	cache map[resourceKey]*serviceInfo
	l     *logger.Logger
}

func servicesURL(ns string) string {
	if ns == "" {
		return "api/v1/services"
	}
	return fmt.Sprintf("api/v1/namespaces/%s/services", ns)
}

func (lister *servicesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	var svcName string
	tok := strings.SplitN(req.GetResourcePath(), "/", 2)
	if len(tok) == 2 {
		svcName = tok[1]
	}

	allFilters, err := filter.ParseFilters(req.GetFilter(), SupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, nsFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.RegexFilters["namespace"], allFilters.LabelsFilter

	lister.mu.RLock()
	defer lister.mu.RUnlock()

	for _, key := range lister.keys {
		if svcName != "" && key.name != svcName {
			continue
		}

		if nameFilter != nil && !nameFilter.Match(key.name, lister.l) {
			continue
		}

		svc := lister.cache[key]
		if nsFilter != nil && !nsFilter.Match(svc.Metadata.Namespace, lister.l) {
			continue
		}
		if labelsFilter != nil && !labelsFilter.Match(svc.Metadata.Labels, lister.l) {
			continue
		}

		resources = append(resources, svc.resources(allFilters.RegexFilters["port"], req.GetIpConfig().GetIpType(), lister.l)...)
	}

	lister.l.Infof("kubernetes.listResources: returning %d services", len(resources))
	return resources, nil
}

type loadBalancerStatus struct {
	Ingress []struct {
		IP       string
		Hostname string
	}
}

type serviceInfo struct {
	Metadata kMetadata
	Spec     struct {
		ClusterIP string
		Ports     []struct {
			Name string
			Port int
		}
	}
	Status struct {
		LoadBalancer loadBalancerStatus
	}
}

func (si *serviceInfo) matchPorts(portFilter *filter.RegexFilter, l *logger.Logger) ([]int, map[int]string) {
	ports, portNameMap := []int{}, make(map[int]string)
	for _, port := range si.Spec.Ports {
		// For unnamed ports, use port number.
		portName := port.Name
		if portName == "" {
			portName = strconv.FormatInt(int64(port.Port), 10)
		}

		if portFilter != nil && !portFilter.Match(portName, l) {
			continue
		}
		ports = append(ports, port.Port)
		portNameMap[port.Port] = portName
	}
	return ports, portNameMap
}

// resources returns RDS resources corresponding to a service resource. Each
// service object can have multiple ports.
//
// a) If service has only 1 port or there is a port filter and only one port
// matches the port filter, we return only one RDS resource with same name as
// service name.
// b) If there are multiple ports, we create one RDS resource for each port and
// name each resource as: <service_name>_<port_name>
func (si *serviceInfo) resources(portFilter *filter.RegexFilter, reqIPType pb.IPConfig_IPType, l *logger.Logger) (resources []*pb.Resource) {
	ports, portNameMap := si.matchPorts(portFilter, l)
	for _, port := range ports {
		resName := si.Metadata.Name
		if len(ports) != 1 {
			resName = fmt.Sprintf("%s_%s", si.Metadata.Name, portNameMap[port])
		}

		res := &pb.Resource{
			Name:   proto.String(resName),
			Port:   proto.Int32(int32(port)),
			Labels: si.Metadata.Labels,
		}

		if reqIPType == pb.IPConfig_PUBLIC {
			// If there is no ingress IP, skip the resource.
			if len(si.Status.LoadBalancer.Ingress) == 0 {
				continue
			}
			ingress := si.Status.LoadBalancer.Ingress[0]

			res.Ip = proto.String(ingress.IP)
			if ingress.IP == "" && ingress.Hostname != "" {
				res.Ip = proto.String(ingress.Hostname)
			}
		} else {
			res.Ip = proto.String(si.Spec.ClusterIP)
		}

		resources = append(resources, res)
	}
	return
}

func parseServicesJSON(resp []byte) (keys []resourceKey, services map[resourceKey]*serviceInfo, err error) {
	var itemList struct {
		Items []*serviceInfo
	}

	if err = json.Unmarshal(resp, &itemList); err != nil {
		return
	}

	keys = make([]resourceKey, len(itemList.Items))
	services = make(map[resourceKey]*serviceInfo)
	for i, item := range itemList.Items {
		keys[i] = resourceKey{item.Metadata.Namespace, item.Metadata.Name}
		services[keys[i]] = item
	}

	return
}

func (lister *servicesLister) expand() {
	resp, err := lister.kClient.getURL(servicesURL(lister.namespace))
	if err != nil {
		lister.l.Warningf("servicesLister.expand(): error while getting services list from API: %v", err)
	}

	keys, services, err := parseServicesJSON(resp)
	if err != nil {
		lister.l.Warningf("servicesLister.expand(): error while parsing services API response (%s): %v", string(resp), err)
	}

	lister.l.Infof("servicesLister.expand(): got %d services", len(keys))

	lister.mu.Lock()
	defer lister.mu.Unlock()
	lister.keys = keys
	lister.cache = services
}

func newServicesLister(c *configpb.Services, namespace string, reEvalInterval time.Duration, kc *client, l *logger.Logger) (*servicesLister, error) {
	lister := &servicesLister{
		c:         c,
		kClient:   kc,
		namespace: namespace,
		l:         l,
	}

	go func() {
		lister.expand()
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// gceInstances, this will make sure that each instance calls GCE
		// API at a different point of time.
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			lister.expand()
		}
	}()

	return lister, nil
}
