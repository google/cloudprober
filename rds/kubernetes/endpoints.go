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

type epLister struct {
	c         *configpb.Endpoints
	namespace string
	kClient   *client

	mu    sync.RWMutex // Mutex for names and cache
	names []string
	cache map[string]*epInfo
	l     *logger.Logger
}

func epURL(ns string) string {
	if ns == "" {
		return "api/v1/endpoints"
	}
	return fmt.Sprintf("api/v1/namespaces/%s/endpoints", ns)
}

func (lister *epLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	var epName string
	tok := strings.SplitN(req.GetResourcePath(), "/", 2)
	if len(tok) == 2 {
		epName = tok[1]
	}

	allFilters, err := filter.ParseFilters(req.GetFilter(), []string{"name", "namespace"}, "")
	if err != nil {
		return nil, err
	}

	nameFilter, nsFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.RegexFilters["namespace"], allFilters.LabelsFilter

	lister.mu.RLock()
	defer lister.mu.RUnlock()

	for _, name := range lister.names {
		if epName != "" && name != epName {
			continue
		}

		if nameFilter != nil && !nameFilter.Match(name, lister.l) {
			continue
		}

		epi := lister.cache[name]
		if nsFilter != nil && !nsFilter.Match(epi.Metadata.Namespace, lister.l) {
			continue
		}
		if labelsFilter != nil && !labelsFilter.Match(epi.Metadata.Labels, lister.l) {
			continue
		}

		resources = append(resources, epi.resources()...)
	}

	lister.l.Infof("kubernetes.endpoints.listResources: returning %d resources", len(resources))
	return resources, nil
}

type epSubset struct {
	Addresses []struct {
		IP string
	}
	Ports []struct {
		Name string
		Port int
	}
}

type epInfo struct {
	Metadata kMetadata
	Subsets  []epSubset
}

// resources returns RDS resources corresponding to an endpoints resource. Each
// endpoints object can have multiple endpoint subsets and each subset in turn
// is composed of multiple addresses and ports. If an endpoint subset as 3
// addresses and 2 ports, there will be 6 resources corresponding to that
// subset.
func (epi *epInfo) resources() (resources []*pb.Resource) {
	for _, eps := range epi.Subsets {
		// There is usually one port, but there can be multiple ports, e.g. 9313
		// and 9314.
		for _, port := range eps.Ports {
			// For unnamed ports, use port number.
			portName := port.Name
			if portName == "" {
				portName = strconv.FormatInt(int64(port.Port), 10)
			}

			for _, addr := range eps.Addresses {
				// We name the resource as <endpoints_name>_<IP>_<port>
				resName := fmt.Sprintf("%s_%s_%s", epi.Metadata.Name, addr.IP, portName)
				resources = append(resources, &pb.Resource{
					Name:   proto.String(resName),
					Ip:     proto.String(addr.IP),
					Port:   proto.Int(port.Port),
					Labels: epi.Metadata.Labels,
				})
			}
		}
	}
	return
}

func parseEndpointsJSON(resp []byte) (names []string, endpoints map[string]*epInfo, err error) {
	var itemList struct {
		Items []*epInfo
	}

	if err = json.Unmarshal(resp, &itemList); err != nil {
		return
	}

	names = make([]string, len(itemList.Items))
	endpoints = make(map[string]*epInfo)
	for i, item := range itemList.Items {
		names[i] = item.Metadata.Name
		endpoints[item.Metadata.Name] = item
	}

	return
}

func (lister *epLister) expand() {
	resp, err := lister.kClient.getURL(epURL(lister.namespace))
	if err != nil {
		lister.l.Warningf("epLister.expand(): error while getting endpoints list from API: %v", err)
	}

	names, endpoints, err := parseEndpointsJSON(resp)
	if err != nil {
		lister.l.Warningf("epLister.expand(): error while parsing endpoints API response (%s): %v", string(resp), err)
	}

	lister.l.Infof("epLister.expand(): got %d endpoints", len(names))

	lister.mu.Lock()
	defer lister.mu.Unlock()
	lister.names = names
	lister.cache = endpoints
}

func newEndpointsLister(c *configpb.Endpoints, namespace string, reEvalInterval time.Duration, kc *client, l *logger.Logger) (*epLister, error) {
	lister := &epLister{
		c:         c,
		namespace: namespace,
		kClient:   kc,
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
