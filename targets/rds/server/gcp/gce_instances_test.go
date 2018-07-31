// Copyright 2017-2018 Google Inc.
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
	"net"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	pb "github.com/google/cloudprober/targets/rds/proto"
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

func (ti *testInstance) computeInstance() []*compute.NetworkInterface {
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

	return cNetIfs
}

var testInstancesData = []*testInstance{
	&testInstance{
		name: "ins1",
		netIf: []*testNetIf{
			&testNetIf{"10.216.0.1", "104.100.143.1", "192.168.1.0/24"},
			&testNetIf{"10.216.1.1", "", ""},
		},
	},
	&testInstance{
		name: "ins2",
		netIf: []*testNetIf{
			&testNetIf{"10.216.0.2", "104.100.143.2", "192.168.2.0/24"},
			&testNetIf{"10.216.1.2", "104.100.143.3", ""},
		},
	},
}

func testLister() *gceInstancesLister {
	// Initialize instanceLister manually for testing, using the
	// testInstances data. Default initialization invokes GCE APIs which we want
	// to avoid.
	lister := &gceInstancesLister{
		cache: make(map[string][]*compute.NetworkInterface),
		l:     &logger.Logger{},
	}
	for _, ti := range testInstancesData {
		lister.cache[ti.name] = ti.computeInstance()
		lister.names = append(lister.names, ti.name)
	}

	return lister
}

func TestInstancesResources(t *testing.T) {
	lister := testLister()

	var testIndex int

	// #################################################################
	// Instances, with first NIC and private IP. Expect no errors.
	// #################################################################
	testListResources(t, nil, nil, lister, testInstancesData, testIndex, "private", false)

	// #################################################################
	// Instances, with first NIC and private IP, but a bad filter.
	// #################################################################
	f := &pb.Filter{
		Key:   proto.String("instance_name"),
		Value: proto.String("ins2"),
	}
	testListResources(t, f, nil, lister, testInstancesData[1:], testIndex, "private", true)

	// #################################################################
	// Instances, matching the name "ins2" and with first NIC and private IP
	// Expect no errors.
	// #################################################################
	f = &pb.Filter{
		Key:   proto.String("name"),
		Value: proto.String("ins2"),
	}
	testListResources(t, f, nil, lister, testInstancesData[1:], testIndex, "private", false)

	// #################################################################
	// Instances, with first NIC and public IP
	// Expect no errors.
	// #################################################################
	ipConfig := &pb.IPConfig{
		IpType: pb.IPConfig_PUBLIC.Enum(),
	}
	testListResources(t, nil, ipConfig, lister, testInstancesData, testIndex, "public", false)

	// #################################################################
	// Instances, with first NIC and alias IP
	// Expect no errors.
	// #################################################################
	ipConfig = &pb.IPConfig{
		IpType: pb.IPConfig_ALIAS.Enum(),
	}
	testListResources(t, nil, ipConfig, lister, testInstancesData, testIndex, "ipAliasRange", false)

	// #################################################################
	// Instances, with second NIC and private IP
	// Expect no errors.
	// #################################################################
	testIndex = 1
	ipConfig = &pb.IPConfig{
		NicIndex: proto.Int32(int32(testIndex)),
	}
	testListResources(t, nil, ipConfig, lister, testInstancesData, testIndex, "private", false)

	// #################################################################
	// Instances, with second NIC and public IP
	// We get an error as first instance's second NIC doesn't have public IP.
	// ##################################################################
	testIndex = 1
	ipConfig = &pb.IPConfig{
		NicIndex: proto.Int32(int32(testIndex)),
		IpType:   pb.IPConfig_PUBLIC.Enum(),
	}
	testListResources(t, nil, ipConfig, lister, nil, testIndex, "public", true)
}

func testListResources(t *testing.T, f *pb.Filter, ipConfig *pb.IPConfig, gil *gceInstancesLister, expectedInstances []*testInstance, testIndex int, ipTypeStr string, expectError bool) {
	var filters []*pb.Filter
	if f != nil {
		filters = append(filters, f)
	}

	resources, err := gil.listResources(filters, ipConfig)

	if err != nil {
		if !expectError {
			t.Errorf("Got error while listing resources: %v, expected no errors", err)
		}
		return
	}

	if len(resources) != len(expectedInstances) {
		t.Errorf("Got wrong number of targets. Expected: %d, Got: %d", len(expectedInstances), len(resources))
	}

	// Build a map from the resources in the reply.
	instances := make(map[string]string)
	for _, res := range resources {
		instances[res.GetName()] = res.GetIp()
	}
	for _, ti := range expectedInstances {
		ip := instances[ti.name]

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
		if ip != expectedIP {
			t.Errorf("Got wrong <%s> IP for %s. Expected: %s, Got: %s", ipTypeStr, ti.name, expectedIP, ip)
		}
	}
}
