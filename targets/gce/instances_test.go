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
	"net"
	"testing"

	"github.com/golang/protobuf/proto"
	configpb "github.com/google/cloudprober/targets/gce/proto"
	compute "google.golang.org/api/compute/v1"
)

type testNetIf struct {
	privateIP    string
	publicIP     string
	aliasIPRange string
}

type testInstance struct {
	name  string
	netIf []*testNetIf
}

func (ti *testInstance) computeInstance() *compute.Instance {
	var cNetIfs []*compute.NetworkInterface
	for _, ni := range ti.netIf {
		cNetIf := &compute.NetworkInterface{
			NetworkIP: ni.privateIP,
		}
		if ni.publicIP != "" {
			cNetIf.AccessConfigs = []*compute.AccessConfig{
				&compute.AccessConfig{
					NatIP: ni.publicIP,
				},
			}
		}
		if ni.aliasIPRange != "" {
			cNetIf.AliasIpRanges = []*compute.AliasIpRange{
				&compute.AliasIpRange{
					IpCidrRange: ni.aliasIPRange,
				},
			}
		}
		cNetIfs = append(cNetIfs, cNetIf)
	}

	return &compute.Instance{
		NetworkInterfaces: cNetIfs,
	}
}

func TestInstancesTargets(t *testing.T) {
	testInstances := []*testInstance{
		&testInstance{
			name: "ins1",
			netIf: []*testNetIf{
				&testNetIf{"10.216.0.1", "104.100.143.1", "192.168.1.0/24"},
			},
		},
		&testInstance{
			name: "ins2",
			netIf: []*testNetIf{
				&testNetIf{"10.216.0.2", "104.100.143.2", "192.168.2.0/24"},
				&testNetIf{"10.216.1.2", "", ""},
			},
		},
	}

	// Initialize globalInstancesProvider manually for testing, using the testInstances
	// data. Default initialization invokes GCE APIs which we want to avoid.
	globalInstancesProvider = &instancesProvider{
		cache: make(map[string]*compute.Instance),
	}
	for _, ti := range testInstances {
		globalInstancesProvider.cache[ti.name] = ti.computeInstance()
		globalInstancesProvider.names = append(globalInstancesProvider.names, ti.name)
	}

	var testIndex int

	// #################################################################
	// Targets, with first NIC and private IP
	// #################################################################
	tgts := &instances{
		pb: &configpb.Instances{},
	}
	testListAndResolve(t, tgts, testInstances, testIndex, "private")

	// #################################################################
	// Targets, with first NIC and public IP
	// #################################################################
	ipType := configpb.Instances_NetworkInterface_PUBLIC
	tgts = &instances{
		pb: &configpb.Instances{
			NetworkInterface: &configpb.Instances_NetworkInterface{
				IpType: &ipType,
			},
		},
	}
	testListAndResolve(t, tgts, testInstances, testIndex, "public")

	// #################################################################
	// Targets, with first NIC and alias IP
	// #################################################################
	ipType = configpb.Instances_NetworkInterface_ALIAS
	tgts = &instances{
		pb: &configpb.Instances{
			NetworkInterface: &configpb.Instances_NetworkInterface{
				IpType: &ipType,
			},
		},
	}
	testListAndResolve(t, tgts, testInstances, testIndex, "ipAliasRange")

	// #################################################################
	// Targets, with second NIC and private IP
	// Resolve will fail for ins1 as it has only one NIC
	// #################################################################
	testIndex = 1
	tgts = &instances{
		pb: &configpb.Instances{
			NetworkInterface: &configpb.Instances_NetworkInterface{
				Index: proto.Int32(int32(testIndex)),
			},
		},
	}
	testListAndResolve(t, tgts, testInstances, testIndex, "private")

	// #################################################################
	// Targets, with second NIC and public IP
	// Resolve will fail for both instances as first one doesn't have
	// second NIC and second instance's second NIC doesn't have public IP
	// ##################################################################
	testIndex = 1
	ipType = configpb.Instances_NetworkInterface_PUBLIC
	tgts = &instances{
		pb: &configpb.Instances{
			NetworkInterface: &configpb.Instances_NetworkInterface{
				Index:  proto.Int32(int32(testIndex)),
				IpType: &ipType,
			},
		},
	}
	testListAndResolve(t, tgts, testInstances, testIndex, "public")
}

func testListAndResolve(t *testing.T, i *instances, testInstances []*testInstance, testIndex int, ipTypeStr string) {
	if len(i.List()) != len(testInstances) {
		t.Errorf("Got wrong number of targets. Expected: %d, Got: %d", len(testInstances), len(i.List()))
	}
	for _, ti := range testInstances {
		ip, err := i.Resolve(ti.name, 4)
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
				t.Errorf("Expected error while resolving for an IP type <%s> that doesn't exist on network interface. Network interface index: %d, Instance name: %s.", ipTypeStr, testIndex, ti.name)
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
