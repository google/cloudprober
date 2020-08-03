// Copyright 2017-2019 Google Inc.
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
	"context"
	"errors"
	"net"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/rds/client"
	rdspb "github.com/google/cloudprober/rds/proto"
	configpb "github.com/google/cloudprober/targets/gce/proto"
)

type testNetIf struct {
	privateIP    string
	publicIP     string
	aliasIPRange string
}

type testInstance struct {
	name   string
	labels map[string]string
	netIf  []*testNetIf
}

var testInstancesDB = map[string][]*testInstance{
	"proj1": []*testInstance{
		&testInstance{
			name: "ins1",
			netIf: []*testNetIf{
				&testNetIf{"10.216.0.1", "104.100.143.1", "192.168.1.0/24"},
			},
			labels: map[string]string{
				"key1": "a",
				"key2": "b",
			},
		},
		&testInstance{
			name: "ins2",
			netIf: []*testNetIf{
				&testNetIf{"10.216.0.2", "104.100.143.2", "192.168.2.0/24"},
				&testNetIf{"10.216.1.2", "", ""},
			},
			labels: map[string]string{
				"key1": "a",
			},
		},
	},
	"proj2": []*testInstance{
		&testInstance{
			name: "ins21",
			labels: map[string]string{
				"key1": "a",
			},
		},
	},
	"proj3": []*testInstance{
		&testInstance{
			name:   "ins31",
			labels: make(map[string]string),
		},
	},
}

func TestInstancesTargets(t *testing.T) {
	var testIndex int

	// #################################################################
	// Targets, with first NIC and private IP
	// #################################################################
	c := &configpb.Instances{}
	testListAndResolve(t, c, testIndex, "private")

	// #################################################################
	// Targets, with first NIC and public IP
	// #################################################################
	ipType := configpb.Instances_NetworkInterface_PUBLIC
	c = &configpb.Instances{
		NetworkInterface: &configpb.Instances_NetworkInterface{
			IpType: &ipType,
		},
	}
	testListAndResolve(t, c, testIndex, "public")

	// #################################################################
	// Targets, with first NIC and alias IP
	// #################################################################
	ipType = configpb.Instances_NetworkInterface_ALIAS
	c = &configpb.Instances{
		NetworkInterface: &configpb.Instances_NetworkInterface{
			IpType: &ipType,
		},
	}
	testListAndResolve(t, c, testIndex, "ipAliasRange")

	// #################################################################
	// Targets, with second NIC and private IP
	// Resolve will fail for ins1 as it has only one NIC
	// #################################################################
	c = &configpb.Instances{
		NetworkInterface: &configpb.Instances_NetworkInterface{
			Index: proto.Int32(int32(testIndex)),
		},
	}
	testListAndResolve(t, c, testIndex, "private")

	// #################################################################
	// Targets, with second NIC and public IP
	// Resolve will fail for both instances as first one doesn't have
	// second NIC and second instance's second NIC doesn't have public IP
	// ##################################################################
	testIndex = 1
	ipType = configpb.Instances_NetworkInterface_PUBLIC
	c = &configpb.Instances{
		NetworkInterface: &configpb.Instances_NetworkInterface{
			Index:  proto.Int32(int32(testIndex)),
			IpType: &ipType,
		},
	}
	testListAndResolve(t, c, testIndex, "public")
}

func testGCEResources(t *testing.T, c *configpb.Instances) *gceResources {
	gr := &gceResources{
		resourceType: "gce_instances",
		clients:      make(map[string]*client.Client),
		ipConfig:     instancesIPConfig(c),
	}

	if len(c.GetLabel()) > 0 {
		filters, err := parseLabels(c.GetLabel())
		if err != nil {
			t.Errorf("Error parsing configured labels: %v", err)
		}
		gr.filters = filters
	}

	for project := range testInstancesDB {
		client, err := newRDSClient(gr.rdsRequest(project), funcListResources, &logger.Logger{})
		if err != nil {
			t.Errorf("Error creating RDS clients for testing: %v", err)
		}
		gr.clients[project] = client
	}

	return gr
}

func testListAndResolve(t *testing.T, c *configpb.Instances, testIndex int, ipTypeStr string) {
	gr := testGCEResources(t, c)

	// Merge instances from all projects to get expectedList
	var expectedList []string
	for _, pTestInstances := range testInstancesDB {
		for _, ti := range pTestInstances {
			expectedList = append(expectedList, ti.name)
		}
	}
	sort.Strings(expectedList)

	var gotList []string
	for _, ep := range gr.ListEndpoints() {
		gotList = append(gotList, ep.Name)
	}
	sort.Strings(gotList)

	if !reflect.DeepEqual(gotList, expectedList) {
		t.Errorf("Got wrong list of targets. Expected: %v, Got: %v", expectedList, gotList)
	}

	// Verify resolve functionality.
	for _, pTestInstances := range testInstancesDB {
		for _, ti := range pTestInstances {
			ip, err := gr.Resolve(ti.name, 4)
			// If instance doesn't have required number of NICs.
			if len(ti.netIf) <= testIndex {
				if err == nil {
					t.Errorf("Expected error while resolving for network interface that doesn't exist. Network interface index: %d, Instance name: %s.", testIndex, ti.name)
				}
				continue
			}
			var expectedIP string
			switch ipTypeStr {
			case "private":
				expectedIP = ti.netIf[testIndex].privateIP
			case "public":
				expectedIP = ti.netIf[testIndex].publicIP
			case "ipAliasRange":
				expectedNetIP, _, _ := net.ParseCIDR(ti.netIf[testIndex].aliasIPRange)
				expectedIP = expectedNetIP.String()
			}
			// Didn't expect an IP address and we didn't get error
			if expectedIP == "" {
				if err == nil {
					t.Errorf("Expected error while resolving for an IP type <%s> that doesn't exist on network interface. Network interface index: %d, Instance name: %s, Got IP: %s", ipTypeStr, testIndex, ti.name, ip.String())
				}
				continue
			}
			if err != nil {
				t.Errorf("Got unexpected error while resolving private IP for %s. Err: %v", ti.name, err)
			}
			if ip.String() != expectedIP {
				t.Errorf("Got wrong <%s> IP for %s. Expected: %s, Got: %s", ipTypeStr, ti.name, expectedIP, ip)
			}
		}
	}
}

// We use funcListResources to create RDS clients for testing purpose.
func funcListResources(ctx context.Context, in *rdspb.ListResourcesRequest) (*rdspb.ListResourcesResponse, error) {
	path := in.GetResourcePath()
	if !strings.HasPrefix(path, "gce_instances/") {
		return nil, errors.New("unsupported resource_type")
	}

	project := strings.TrimPrefix(path, "gce_instances/")
	var resources []*rdspb.Resource

	for _, ti := range testInstancesDB[project] {
		res := &rdspb.Resource{Name: &ti.name}

		// We do a minimal labels check for testing purpose. Detailed tests are
		// done within the RDS package.
		matched := true
		for _, f := range in.GetFilter() {
			if strings.HasPrefix(f.GetKey(), "labels.") {
				if _, ok := ti.labels[strings.TrimPrefix(f.GetKey(), "labels.")]; !ok {
					matched = false
				}
			}
		}
		if !matched {
			continue
		}

		if len(ti.netIf) < int(in.GetIpConfig().GetNicIndex())+1 {
			res.Ip = proto.String("")
		} else {
			ni := ti.netIf[in.GetIpConfig().GetNicIndex()]
			var ip string

			switch in.GetIpConfig().GetIpType().String() {
			case "DEFAULT":
				ip = ni.privateIP
			case "PUBLIC":
				ip = ni.publicIP
			case "ALIAS":
				ip1, _, _ := net.ParseCIDR(ni.aliasIPRange)
				ip = ip1.String()
			}

			res.Ip = &ip
		}

		resources = append(resources, res)
	}

	return &rdspb.ListResourcesResponse{Resources: resources}, nil
}

func TestIPListFilteringByLabels(t *testing.T) {
	tests := []struct {
		desc  string
		label string
		want  []string
	}{
		{
			"key1",
			"key1:a",
			[]string{"ins1", "ins2", "ins21"},
		},
		{
			"key2",
			"key2:a",
			[]string{"ins1"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			c := &configpb.Instances{
				Label: []string{test.label},
			}
			gr := testGCEResources(t, c)

			var gotList []string
			for _, ep := range gr.ListEndpoints() {
				gotList = append(gotList, ep.Name)
			}
			sort.Strings(gotList)

			if !reflect.DeepEqual(gotList, test.want) {
				t.Errorf("Got wrong list of targets. Got: %v, Want: %v", gotList, test.want)
			}
		})
	}
}
