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
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/kubernetes/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/filter"
)

type podsLister struct {
	c       *configpb.Pods
	kClient *client

	mu    sync.RWMutex // Mutex for names and cache
	names []string
	cache map[string]*resource
	l     *logger.Logger
}

func podsURL(ns string) string {
	if ns == "" {
		return "api/v1/pods"
	}
	return fmt.Sprintf("api/v1/namespaces/%s/pods", ns)
}

// ipFunc implements the logic for retrieving IP address from a pod's JSON
// representation.
func podIPFunc(item jsonItem, l *logger.Logger) (string, error) {
	var podStatus struct {
		Phase string
		PodIP string
	}

	err := json.Unmarshal(item.Status, &podStatus)
	if err != nil {
		return "", fmt.Errorf("error parsing status for the pod: %s", item.Metadata.Name)
	}

	if podStatus.Phase != "Running" {
		l.Infof("Ignoring getting IP for non-running pod: %s", item.Metadata.Name)
		return "", nil
	}

	return podStatus.PodIP, nil
}

func (pl *podsLister) listResources(filters []*pb.Filter) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	allFilters, err := filter.ParseFilters(filters, []string{"name", "namespace"}, "")
	if err != nil {
		return nil, err
	}

	nameFilter, nsFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.RegexFilters["namespace"], allFilters.LabelsFilter

	pl.mu.RLock()
	defer pl.mu.RUnlock()

	for _, name := range pl.names {
		if nameFilter != nil && !nameFilter.Match(name, pl.l) {
			continue
		}

		res := pl.cache[name]
		if nsFilter != nil && !nsFilter.Match(res.namespace, pl.l) {
			continue
		}
		if labelsFilter != nil && !labelsFilter.Match(res.labels, pl.l) {
			continue
		}

		resources = append(resources, &pb.Resource{
			Name:   proto.String(name),
			Ip:     proto.String(res.ip),
			Port:   proto.Int32(int32(res.port)),
			Labels: res.labels,
		})
	}

	pl.l.Infof("kubernetes.listResources: returning %d pods", len(resources))
	return resources, nil
}

func (pl *podsLister) expand() {
	resp, err := pl.kClient.getURL(podsURL(pl.c.GetNamespace()))
	if err != nil {
		pl.l.Warningf("podsLister.expand(): error while getting pods list from API: %v", err)
	}

	names, resources, err := parseResourceList(resp, func(item jsonItem) (string, error) { return podIPFunc(item, pl.l) })
	if err != nil {
		pl.l.Warningf("podsLister.expand(): error while parsing pods API response (%s): %v", string(resp), err)
	}

	pl.l.Infof("podsLister.expand(): got %d pods", len(names))

	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.names = names
	pl.cache = resources
}

func newPodsLister(c *configpb.Pods, kc *client, l *logger.Logger) (*podsLister, error) {
	pl := &podsLister{
		c:       c,
		kClient: kc,
		l:       l,
	}

	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second
	go func() {
		pl.expand()
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// gceInstances, this will make sure that each instance calls GCE
		// API at a different point of time.
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			pl.expand()
		}
	}()

	return pl, nil
}
