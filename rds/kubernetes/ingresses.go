// Copyright 2020 The Cloudprober Authors.
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
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/kubernetes/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/filter"
)

type ingressesLister struct {
	c         *configpb.Ingresses
	namespace string
	kClient   *client

	mu    sync.RWMutex // Mutex for names and cache
	keys  []resourceKey
	cache map[resourceKey]*ingressInfo
	l     *logger.Logger
}

func ingressesURL(ns string) string {
	// TODO(manugarg): Update version to v1 once it's more widely available.
	if ns == "" {
		return "apis/networking.k8s.io/v1beta1/ingresses"
	}
	return fmt.Sprintf("apis/networking.k8s.io/v1beta1/namespaces/%s/ingresses", ns)
}

func (lister *ingressesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	var resName string
	tok := strings.SplitN(req.GetResourcePath(), "/", 2)
	if len(tok) == 2 {
		resName = tok[1]
	}

	allFilters, err := filter.ParseFilters(req.GetFilter(), SupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, nsFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.RegexFilters["namespace"], allFilters.LabelsFilter

	lister.mu.RLock()
	defer lister.mu.RUnlock()

	for _, key := range lister.keys {
		if resName != "" && key.name != resName {
			continue
		}

		ingress := lister.cache[key]
		if nsFilter != nil && !nsFilter.Match(ingress.Metadata.Namespace, lister.l) {
			continue
		}

		for _, res := range ingress.resources() {
			if nameFilter != nil && !nameFilter.Match(res.GetName(), lister.l) {
				continue
			}
			if labelsFilter != nil && !labelsFilter.Match(res.GetLabels(), lister.l) {
				continue
			}
			resources = append(resources, res)
		}
	}

	lister.l.Infof("kubernetes.listResources: returning %d ingresses", len(resources))
	return resources, nil
}

type ingressRule struct {
	Host string
	HTTP struct {
		Paths []struct {
			Path string
		}
	}
}

type ingressInfo struct {
	Metadata kMetadata
	Spec     struct {
		Rules []ingressRule
	}
	Status struct {
		LoadBalancer loadBalancerStatus
	}
}

// resources returns RDS resources corresponding to an ingress resource.
func (i *ingressInfo) resources() (resources []*pb.Resource) {
	resName := i.Metadata.Name
	baseLabels := i.Metadata.Labels

	// Note that for ingress we don't check the type of the IP in the request.
	// That is mainly because ingresses typically have only one ingress
	// controller and hence one IP address. Also, the difference of private vs
	// public IP doesn't really exist.
	var ip string
	if len(i.Status.LoadBalancer.Ingress) > 0 {
		ii := i.Status.LoadBalancer.Ingress[0]
		ip = ii.IP
		if ip == "" && ii.Hostname != "" {
			ip = ii.Hostname
		}
	}

	if len(i.Spec.Rules) == 0 {
		return []*pb.Resource{
			{
				Name:   proto.String(resName),
				Labels: baseLabels,
				Ip:     proto.String(ip),
			},
		}
	}

	for _, rule := range i.Spec.Rules {
		nameWithHost := fmt.Sprintf("%s_%s", resName, rule.Host)

		for _, p := range rule.HTTP.Paths {
			nameWithPath := nameWithHost
			if p.Path != "/" {
				nameWithPath = fmt.Sprintf("%s_%s", nameWithHost, strings.Replace(p.Path, "/", "_", -1))
			}

			// Add fqdn and url labels to the resources.
			labels := make(map[string]string, len(baseLabels)+2)
			for k, v := range baseLabels {
				labels[k] = v
			}
			if _, ok := labels["fqdn"]; !ok {
				labels["fqdn"] = rule.Host
			}
			if _, ok := labels["relative_url"]; !ok {
				labels["relative_url"] = p.Path
			}

			resources = append(resources, &pb.Resource{
				Name:   proto.String(nameWithPath),
				Labels: labels,
				Ip:     proto.String(ip),
			})
		}
	}

	return
}

func parseIngressesJSON(resp []byte) (keys []resourceKey, ingresses map[resourceKey]*ingressInfo, err error) {
	var itemList struct {
		Items []*ingressInfo
	}

	if err = json.Unmarshal(resp, &itemList); err != nil {
		return
	}

	keys = make([]resourceKey, len(itemList.Items))
	ingresses = make(map[resourceKey]*ingressInfo)
	for i, item := range itemList.Items {
		keys[i] = resourceKey{item.Metadata.Namespace, item.Metadata.Name}
		ingresses[keys[i]] = item
	}

	return
}

func (lister *ingressesLister) expand() {
	resp, err := lister.kClient.getURL(ingressesURL(lister.namespace))
	if err != nil {
		lister.l.Warningf("ingressesLister.expand(): error while getting ingresses list from API: %v", err)
	}

	keys, ingresses, err := parseIngressesJSON(resp)
	if err != nil {
		lister.l.Warningf("ingressesLister.expand(): error while parsing ingresses API response (%s): %v", string(resp), err)
	}

	lister.l.Infof("ingressesLister.expand(): got %d ingresses", len(keys))

	lister.mu.Lock()
	defer lister.mu.Unlock()
	lister.keys = keys
	lister.cache = ingresses
}

func newIngressesLister(c *configpb.Ingresses, namespace string, reEvalInterval time.Duration, kc *client, l *logger.Logger) (*ingressesLister, error) {
	lister := &ingressesLister{
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
