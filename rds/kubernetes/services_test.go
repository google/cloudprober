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
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/google/cloudprober/rds/proto"
)

func testServiceInfo(name, ns, ip, publicIP string, labels map[string]string, ports []int) *serviceInfo {
	si := &serviceInfo{Metadata: kMetadata{Name: name, Namespace: ns, Labels: labels}}
	si.Spec.ClusterIP = ip

	for _, port := range ports {
		si.Spec.Ports = append(si.Spec.Ports, struct {
			Name string
			Port int
		}{
			Name: "",
			Port: port,
		})
	}

	if publicIP != "" {
		si.Status.LoadBalancer.Ingress = []struct{ IP string }{
			{
				IP: publicIP,
			},
		}
	}
	return si
}

func TestListSvcResources(t *testing.T) {
	sl := &servicesLister{}
	sl.names = []string{"serviceA", "serviceB", "serviceC"}
	sl.cache = map[string]*serviceInfo{
		"serviceA": testServiceInfo("serviceA", "nsAB", "10.1.1.1", "", map[string]string{"app": "appA"}, []int{9313, 9314}),
		"serviceB": testServiceInfo("serviceB", "nsAB", "10.1.1.2", "192.16.16.199", map[string]string{"app": "appB"}, []int{443}),
		"serviceC": testServiceInfo("serviceC", "nsC", "10.1.1.3", "192.16.16.200", map[string]string{"app": "appC", "func": "web"}, []int{3141}),
	}

	tests := []struct {
		desc          string
		nameFilter    string
		filters       map[string]string
		labelsFilter  map[string]string
		wantServices  []string
		wantIPs       []string
		wantPorts     []int32
		wantPublicIPs []string
		wantErr       bool
	}{
		{
			desc:    "bad filter key, expect error",
			filters: map[string]string{"names": "service(B|C)"},
			wantErr: true,
		},
		{
			desc:         "only name filter for serviceB and serviceC",
			filters:      map[string]string{"name": "service(B|C)"},
			wantServices: []string{"serviceB", "serviceC"},
			wantIPs:      []string{"10.1.1.2", "10.1.1.3"},
			wantPorts:    []int32{443, 3141},
		},
		{
			desc:         "only port filter for serviceA and serviceB's ports 9314 and 3141",
			filters:      map[string]string{"port": "314"},
			wantServices: []string{"serviceA", "serviceC"},
			wantIPs:      []string{"10.1.1.1", "10.1.1.3"},
			wantPorts:    []int32{9314, 3141},
		},
		{
			desc:         "name and namespace filter for serviceB",
			filters:      map[string]string{"name": "service(B|C)", "namespace": "nsAB"},
			wantServices: []string{"serviceB"},
			wantIPs:      []string{"10.1.1.2"},
			wantPorts:    []int32{443},
		},
		{
			desc:         "only namespace filter for serviceA and serviceB",
			filters:      map[string]string{"namespace": "nsAB"},
			wantServices: []string{"serviceA_9313", "serviceA_9314", "serviceB"},
			wantIPs:      []string{"10.1.1.1", "10.1.1.1", "10.1.1.2"},
			wantPorts:    []int32{9313, 9314, 443},
		},
		{
			desc:          "only services with public IPs",
			wantServices:  []string{"serviceB", "serviceC"},
			wantPublicIPs: []string{"192.16.16.199", "192.16.16.200"},
			wantPorts:     []int32{443, 3141},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var filtersPB []*pb.Filter
			for k, v := range test.filters {
				filtersPB = append(filtersPB, &pb.Filter{Key: proto.String(k), Value: proto.String(v)})
			}

			req := &pb.ListResourcesRequest{Filter: filtersPB}

			if len(test.wantPublicIPs) != 0 {
				req.IpConfig = &pb.IPConfig{
					IpType: pb.IPConfig_PUBLIC.Enum(),
				}
			}

			results, err := sl.listResources(req)
			if err != nil {
				if !test.wantErr {
					t.Errorf("got unexpected error: %v", err)
				}
				return
			}

			var gotNames, gotIPs []string
			var gotPorts []int32
			for _, res := range results {
				gotNames = append(gotNames, res.GetName())
				gotIPs = append(gotIPs, res.GetIp())
				gotPorts = append(gotPorts, res.GetPort())
			}

			if !reflect.DeepEqual(gotNames, test.wantServices) {
				t.Errorf("services.listResources: got=%v, expected=%v", gotNames, test.wantServices)
			}

			wantIPs := test.wantIPs
			if len(test.wantPublicIPs) != 0 {
				wantIPs = test.wantPublicIPs
			}

			if !reflect.DeepEqual(gotIPs, wantIPs) {
				t.Errorf("services.listResources IPs: got=%v, expected=%v", gotIPs, wantIPs)
			}

			if !reflect.DeepEqual(gotPorts, test.wantPorts) {
				t.Errorf("services.listResources Ports: got=%v, expected=%v", gotPorts, test.wantPorts)
			}
		})
	}
}

func TestParseSvcResourceList(t *testing.T) {
	servicesListFile := "./testdata/services.json"
	data, err := ioutil.ReadFile(servicesListFile)

	if err != nil {
		t.Fatalf("error reading test data file: %s", servicesListFile)
	}
	_, servicesByName, err := parseServicesJSON(data)

	if err != nil {
		t.Fatalf("Error while parsing services JSON data: %v", err)
	}

	services := map[string]struct {
		ip       string
		publicIP string
		ports    []int
		labels   map[string]string
	}{
		"cloudprober": {
			ip:     "10.31.252.209",
			ports:  []int{9313},
			labels: map[string]string{"app": "cloudprober"},
		},
		"cloudprober-rds": {
			ip:       "10.96.15.88",
			publicIP: "192.88.99.199",
			ports:    []int{9314, 9313},
			labels:   map[string]string{"app": "cloudprober"},
		},
		"cloudprober-test": {
			ip:     "10.31.246.77",
			ports:  []int{9313},
			labels: map[string]string{"app": "cloudprober"},
		},
		"kubernetes": {
			ip:     "10.31.240.1",
			ports:  []int{443},
			labels: map[string]string{"component": "apiserver", "provider": "kubernetes"},
		},
	}

	for name, svc := range services {
		if servicesByName[name] == nil {
			t.Errorf("didn't get service by the name: %s", name)
		}

		gotLabels := servicesByName[name].Metadata.Labels
		if !reflect.DeepEqual(gotLabels, svc.labels) {
			t.Errorf("%s service labels: got=%v, want=%v", name, gotLabels, svc.labels)
		}

		if servicesByName[name].Spec.ClusterIP != svc.ip {
			t.Errorf("%s service ip: got=%s, want=%s", name, servicesByName[name].Spec.ClusterIP, svc.ip)
		}

		if svc.publicIP != "" {
			if servicesByName[name].Status.LoadBalancer.Ingress[0].IP != svc.publicIP {
				t.Errorf("%s service load balancer ip: got=%s, want=%s", name, servicesByName[name].Status.LoadBalancer.Ingress[0].IP, svc.publicIP)
			}
		}

		var gotPorts []int
		for _, port := range servicesByName[name].Spec.Ports {
			gotPorts = append(gotPorts, port.Port)
		}
		if !reflect.DeepEqual(gotPorts, svc.ports) {
			t.Errorf("%s service ports: got=%v, want=%v", name, gotPorts, svc.ports)
		}
	}
}
