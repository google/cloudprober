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

package kubernetes

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/google/cloudprober/rds/proto"
)

func testIngressInfo(k resourceKey, ingressIP, hostname string, labels map[string]string) *ingressInfo {
	si := &ingressInfo{Metadata: kMetadata{Name: k.name, Namespace: k.namespace, Labels: labels}}

	if ingressIP != "" || hostname != "" {
		si.Status.LoadBalancer.Ingress = []struct{ IP, Hostname string }{
			{
				IP:       ingressIP,
				Hostname: hostname,
			},
		}
	}
	return si
}

func listerFromDataFile(t *testing.T) *ingressesLister {
	t.Helper()

	ingressesListFile := "./testdata/ingresses.json"
	data, err := ioutil.ReadFile(ingressesListFile)

	if err != nil {
		t.Fatalf("error reading test data file: %s", ingressesListFile)
	}
	keys, ingresses, err := parseIngressesJSON(data)

	if err != nil {
		t.Fatalf("Error while parsing ingresses JSON data: %v", err)
	}

	return &ingressesLister{
		keys:  keys,
		cache: ingresses,
	}
}

func TestParseIngressJSON(t *testing.T) {
	lister := listerFromDataFile(t)

	key := resourceKey{"default", "rds-ingress"}
	if len(lister.keys) != 1 || lister.cache[key] == nil {
		t.Errorf("Expected exactly one ingress with name rds-ingress, got: %+v", lister.cache)
	}
}

func TestListIngressResources(t *testing.T) {
	lister := listerFromDataFile(t)

	// Add one more ingress for better testing.
	key := resourceKey{name: "cloudprober-ingress", namespace: "monitoring"}
	lister.keys = append(lister.keys, key)
	lister.cache[key] = testIngressInfo(key, "", "cloudprober.monitoring.com", map[string]string{})

	tests := []struct {
		desc      string
		filters   map[string]string
		wantNames []string
		wantFQDNs []string
		wantURLs  []string
		wantIPs   []string
	}{
		{
			desc:      "no filter",
			wantNames: []string{"rds-ingress_foo.bar.com__health", "rds-ingress_foo.bar.com__rds", "rds-ingress_prometheus.bar.com", "cloudprober-ingress"},
			wantFQDNs: []string{"foo.bar.com", "foo.bar.com", "prometheus.bar.com", ""},
			wantURLs:  []string{"/health", "/rds", "/", ""},
			wantIPs:   []string{"241.120.51.35", "241.120.51.35", "241.120.51.35", "cloudprober.monitoring.com"},
		},
		{
			desc:      "namespace filter",
			filters:   map[string]string{"namespace": "monitoring"},
			wantNames: []string{"cloudprober-ingress"},
			wantFQDNs: []string{""},
			wantURLs:  []string{""},
			wantIPs:   []string{"cloudprober.monitoring.com"},
		},
		{
			desc:      "name filter for host regex",
			filters:   map[string]string{"name": ".*foo.bar.com.*"},
			wantNames: []string{"rds-ingress_foo.bar.com__health", "rds-ingress_foo.bar.com__rds"},
			wantFQDNs: []string{"foo.bar.com", "foo.bar.com"},
			wantURLs:  []string{"/health", "/rds"},
			wantIPs:   []string{"241.120.51.35", "241.120.51.35"},
		},
		{
			desc:      "name and label filter",
			filters:   map[string]string{"name": ".*foo.bar.com.*", "labels.relative_url": "/rds"},
			wantNames: []string{"rds-ingress_foo.bar.com__rds"},
			wantFQDNs: []string{"foo.bar.com"},
			wantURLs:  []string{"/rds"},
			wantIPs:   []string{"241.120.51.35"},
		},
		{
			desc:      "fqdn filter",
			filters:   map[string]string{"labels.fqdn": "prometheus.bar.com"},
			wantNames: []string{"rds-ingress_prometheus.bar.com"},
			wantFQDNs: []string{"prometheus.bar.com"},
			wantURLs:  []string{"/"},
			wantIPs:   []string{"241.120.51.35"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var filters []*pb.Filter

			for k, v := range test.filters {
				filters = append(filters, &pb.Filter{
					Key:   proto.String(k),
					Value: proto.String(v),
				})
			}

			resources, err := lister.listResources(&pb.ListResourcesRequest{Filter: filters})
			if err != nil {
				t.Errorf("Error while listing resources: %v", err)
			}
			var gotNames, gotFQDNs, gotURLs, gotIPs []string
			for _, res := range resources {
				gotNames = append(gotNames, res.GetName())
				gotFQDNs = append(gotFQDNs, res.GetLabels()["fqdn"])
				gotURLs = append(gotURLs, res.GetLabels()["relative_url"])
				gotIPs = append(gotIPs, res.GetIp())
			}

			if !reflect.DeepEqual(gotNames, test.wantNames) {
				t.Errorf("gotName: %v, wantNames: %v", gotNames, test.wantNames)
			}

			if !reflect.DeepEqual(gotFQDNs, test.wantFQDNs) {
				t.Errorf("gotFQDNs: %v, wantFQDNs: %v", gotFQDNs, test.wantFQDNs)
			}

			if !reflect.DeepEqual(gotURLs, test.wantURLs) {
				t.Errorf("gotURLs: %v, wantURLs: %v", gotURLs, test.wantURLs)
			}

			if !reflect.DeepEqual(gotIPs, test.wantIPs) {
				t.Errorf("gotIPs: %v, wantIPs: %v", gotIPs, test.wantIPs)
			}
		})
	}
}
