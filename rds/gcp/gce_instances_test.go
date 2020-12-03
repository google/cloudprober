// Copyright 2017-2020 Google Inc.
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
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	pb "github.com/google/cloudprober/rds/proto"
)

type testNetIf struct {
	privateIP    string
	publicIP     string
	aliasIPRange string
	ipv6         string
	publicIPv6   string
}

type testInstance struct {
	name   string
	zone   string
	netIf  []*testNetIf
	labels map[string]string
}

func (ti *testInstance) data() *instanceData {
	var cNetIfs []networkInterface
	for _, ni := range ti.netIf {
		cNetIf := networkInterface{
			NetworkIP:   ni.privateIP,
			Ipv6Address: ni.ipv6,
		}
		if ni.publicIP != "" {
			cNetIf.AccessConfigs = []accessConfig{
				{NatIP: ni.publicIP},
			}
		}
		if ni.publicIPv6 != "" {
			cNetIf.Ipv6AccessConfigs = []accessConfig{
				{ExternalIpv6: ni.publicIPv6},
			}
		}
		if ni.aliasIPRange != "" {
			cNetIf.AliasIPRanges = []struct {
				IPCidrRange string `json:"ipCidrRange,omitempty"`
			}{
				{
					IPCidrRange: ni.aliasIPRange,
				},
			}
		}
		cNetIfs = append(cNetIfs, cNetIf)
	}

	return &instanceData{ii: &instanceInfo{Name: ti.name, NetworkInterfaces: cNetIfs, Labels: ti.labels}}
}

func (ti *testInstance) expectedIP(spec *testSpec) string {
	switch spec.ipType {
	case "private":
		if spec.ipv6 {
			return ti.netIf[spec.index].ipv6
		}
		return ti.netIf[spec.index].privateIP
	case "public":
		if spec.ipv6 {
			return ti.netIf[spec.index].publicIPv6
		}
		return ti.netIf[spec.index].publicIP
	case "ipAliasRange":
		netIP, _, _ := net.ParseCIDR(ti.netIf[spec.index].aliasIPRange)
		if netIP == nil {
			netIP = net.ParseIP(ti.netIf[spec.index].aliasIPRange)
		}
		return netIP.String()
	}
	return ""
}

var testInstancesData = []*testInstance{
	{
		name: "ins1",
		zone: "z-1",
		netIf: []*testNetIf{
			&testNetIf{"10.216.0.1", "104.100.143.1", "192.168.1.0/24", "2600:2d00:4030:a47:c0a8:2110:0:0", "2600:2d00:4030:a47:c0a8:2110:1:0"},
			&testNetIf{"10.216.1.1", "", "", "", ""},
		},
		labels: map[string]string{
			"env":  "staging",
			"func": "rds",
		},
	},
	{
		name: "ins2",
		zone: "z-2",
		netIf: []*testNetIf{
			&testNetIf{"10.216.0.2", "104.100.143.2", "192.168.2.0", "", ""},
			&testNetIf{"10.216.1.2", "104.100.143.3", "", "", ""},
		},
		labels: map[string]string{
			"env":  "prod",
			"func": "rds",
		},
	},
	{
		name: "ins3",
		zone: "z-2",
		netIf: []*testNetIf{
			&testNetIf{"10.216.0.3", "104.100.143.4", "192.168.3.0", "", ""},
			&testNetIf{"10.216.1.3", "104.100.143.5", "", "", ""},
		},
		labels: map[string]string{
			"env":  "prod",
			"func": "rds",
		},
	},
}

func testLister() *gceInstancesLister {
	// Initialize instanceLister manually for testing, using the
	// testInstances data. Default initialization invokes GCE APIs which we want
	// to avoid.
	lister := &gceInstancesLister{
		cachePerScope: make(map[string]map[string]*instanceData),
		namesPerScope: make(map[string][]string),
		l:             &logger.Logger{},
	}

	for _, ti := range testInstancesData {
		if lister.cachePerScope[ti.zone] == nil {
			lister.cachePerScope[ti.zone] = make(map[string]*instanceData)
		}
		lister.cachePerScope[ti.zone][ti.name] = ti.data()
		lister.namesPerScope[ti.zone] = append(lister.namesPerScope[ti.zone], ti.name)
	}

	return lister
}

type testSpec struct {
	f      []*pb.Filter
	index  int
	ipType string
	ipv6   bool
	err    bool
}

func (ts *testSpec) ipConfig() *pb.IPConfig {
	ipConfig := &pb.IPConfig{}
	if ts.ipv6 {
		ipConfig.IpVersion = pb.IPConfig_IPV6.Enum()
	}
	switch ts.ipType {
	case "public":
		ipConfig.IpType = pb.IPConfig_PUBLIC.Enum()
	case "ipAliasRange":
		ipConfig.IpType = pb.IPConfig_ALIAS.Enum()
	}
	if ts.index != 0 {
		ipConfig.NicIndex = proto.Int32(int32(ts.index))
	}
	return ipConfig
}

func testListResources(t *testing.T, gil *gceInstancesLister, expectedInstances []*testInstance, spec *testSpec) {
	t.Helper()

	if spec == nil {
		spec = &testSpec{}
	}
	if spec.ipType == "" {
		spec.ipType = "private"
	}

	var filters []*pb.Filter
	if spec.f != nil {
		filters = append(filters, spec.f...)
	}

	resources, err := gil.listResources(&pb.ListResourcesRequest{
		Filter:   filters,
		IpConfig: spec.ipConfig(),
	})

	if err != nil {
		if !spec.err {
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
		expectedIP := ti.expectedIP(spec)
		if ip != expectedIP {
			t.Errorf("Got wrong <%s> IP for %s. Expected: %s, Got: %s", spec.ipType, ti.name, expectedIP, ip)
		}
	}
}

func TestInstancesResourcesFilters(t *testing.T) {
	lister := testLister()

	// #################################################################
	// Instances, with first NIC and private IP. Expect no errors.
	// #################################################################
	testListResources(t, lister, testInstancesData, nil)

	// #################################################################
	// Instances, with first NIC and private IP, but a bad filter.
	// #################################################################
	f := []*pb.Filter{
		{
			Key:   proto.String("instance_name"),
			Value: proto.String("ins2"),
		},
	}
	testListResources(t, lister, testInstancesData[1:], &testSpec{f: f, err: true})

	// #################################################################
	// Instances, matching the name "ins." and labels: env:prod, func:rds
	// Expect: all instances, no errors.
	// #################################################################
	f = []*pb.Filter{
		{
			Key:   proto.String("name"),
			Value: proto.String("ins."),
		},
		{
			Key:   proto.String("labels.func"),
			Value: proto.String("rds"),
		},
	}
	testListResources(t, lister, testInstancesData, &testSpec{f: f})

	// #################################################################
	// Instances, matching the name "ins." and labels: env:prod, func:rds
	// Expect: Only second instance, no errors.
	// #################################################################
	f = []*pb.Filter{
		{
			Key:   proto.String("name"),
			Value: proto.String("ins."),
		},
		{
			Key:   proto.String("labels.env"),
			Value: proto.String("prod"),
		},
		{
			Key:   proto.String("labels.func"),
			Value: proto.String("rds"),
		},
	}
	testListResources(t, lister, testInstancesData[1:], &testSpec{f: f})

	// #################################################################
	// Instances, matching the name "ins." and labels: env:prod, func:server
	// Expect: no instance, no errors.
	// #################################################################
	f = []*pb.Filter{
		{
			Key:   proto.String("name"),
			Value: proto.String("ins."),
		},
		{
			Key:   proto.String("labels.env"),
			Value: proto.String("prod"),
		},
		{
			Key:   proto.String("labels.func"),
			Value: proto.String("server"),
		},
	}
	testListResources(t, lister, []*testInstance{}, &testSpec{f: f})
}

func TestInstancesResources(t *testing.T) {
	lister := testLister()

	// #################################################################
	// Instances, with first NIC and private IP. Expect no errors.
	// #################################################################
	testListResources(t, lister, testInstancesData, nil)

	// #################################################################
	// Instances, with first NIC and IPv6
	// Expect no errors.
	// #################################################################
	testListResources(t, lister, testInstancesData, &testSpec{ipv6: true})

	// #################################################################
	// Instances, with first NIC and public IP
	// Expect no errors.
	// #################################################################
	testListResources(t, lister, testInstancesData, &testSpec{ipType: "public"})

	// #################################################################
	// Instances (only ins1), with first NIC, IPv6 and public IP.
	// Expect no errors.
	// #################################################################
	f := []*pb.Filter{
		{
			Key:   proto.String("name"),
			Value: proto.String("ins1"),
		},
	}
	testListResources(t, lister, testInstancesData[:1], &testSpec{f: f, ipType: "public", ipv6: true})

	// #################################################################
	// Instances, with first NIC and alias IP
	// Expect no errors.
	// #################################################################
	testListResources(t, lister, testInstancesData, &testSpec{ipType: "ipAliasRange"})

	// #################################################################
	// Instances, with second NIC and private IP
	// Expect no errors.
	// #################################################################
	testListResources(t, lister, testInstancesData, &testSpec{index: 1})

	// #################################################################
	// Instances, with second NIC and public IP
	// We get an error as first instance's second NIC doesn't have public IP.
	// ##################################################################
	testListResources(t, lister, nil, &testSpec{index: 1, ipType: "public", err: true})
}

func readTestJSON(t *testing.T, fileName string) (b []byte) {
	t.Helper()

	instancesListFile := "./testdata/" + fileName
	data, err := ioutil.ReadFile(instancesListFile)

	if err != nil {
		t.Fatalf("error reading test data file: %s", instancesListFile)
	}

	return data
}

func TestExpand(t *testing.T) {
	project, zone, apiVersion := "proj1", "us-central1-a", "v1"

	testGetURL := func(_ *http.Client, url string) ([]byte, error) {
		switch url {
		case "https://www.googleapis.com/compute/v1/projects/proj1/zones/us-central1-a/instances?filter=status%20eq%20%22RUNNING%22":
			return readTestJSON(t, "instances.json"), nil
		case "https://www.googleapis.com/compute/v1/projects/proj1/zones/us-central1-b/instances?filter=status%20eq%20%22RUNNING%22":
			return []byte("{\"items\": []}"), nil // No instances in us-central1-b
		case "https://www.googleapis.com/compute/v1/projects/proj1/zones":
			return readTestJSON(t, "zones.json"), nil
		}
		// Return error for non-matching URL.
		return nil, fmt.Errorf("unknown url: %s", url)
	}

	il := &gceInstancesLister{
		project:       project,
		baseAPIPath:   "https://www.googleapis.com/compute/" + apiVersion + "/projects/" + project,
		getURLFunc:    testGetURL,
		cachePerScope: make(map[string]map[string]*instanceData),
		namesPerScope: make(map[string][]string),
	}

	timeBeforeExpand := time.Now().Unix()

	il.expand(time.Second)

	var gotZones []string
	for z := range il.namesPerScope {
		gotZones = append(gotZones, z)
	}
	// Sort as zones are shuffled in expand.
	sort.Strings(gotZones)
	wantZones := []string{"us-central1-a", "us-central1-b"}
	if !reflect.DeepEqual(gotZones, wantZones) {
		t.Errorf("got zones=%v, want zones=%v", gotZones, wantZones)
	}

	gotNames, gotInstances := il.namesPerScope[zone], il.cachePerScope[zone]

	// Expected data comes from the JSON file.
	wantNames := []string{
		"ig-us-central1-a-00-abcd",
		"ig-us-central1-a-01-efgh",
	}
	wantLabels := []map[string]string{
		map[string]string{"app": "cloudprober", "shard": "00"},
		map[string]string{"app": "cloudprober", "shard": "01"},
	}
	wantNetworks := [][][4]string{
		[][4]string{
			{"10.0.0.2", "194.197.208.201", "2600:2d00:4030:a47:c0a8:2110:0:0", "2600:2d00:4030:a47:c0a8:2110:1:0"},
			{"10.0.0.3", "194.197.208.202", "2600:2d00:4030:a47:c0a8:2110:0:1", ""},
		},
		[][4]string{{"10.0.1.3", "194.197.209.202", "", ""}},
	}

	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Errorf("Got names=%v, want=%v", gotNames, wantNames)
	}

	for i, name := range wantNames {
		ins := gotInstances[name]

		// Check lastupdate timestamp
		if ins.lastUpdated < timeBeforeExpand {
			t.Errorf("Instance's last update timestamp (%d) was not updated during expand (timestamp before expand: %d).", ins.lastUpdated, timeBeforeExpand)
		}

		// Check for labels
		gotLabels := ins.ii.Labels
		wantLabels := wantLabels[i]
		if !reflect.DeepEqual(gotLabels, wantLabels) {
			t.Errorf("Got labels=%v, want labels=%v", gotLabels, wantLabels)
		}

		// Check for ips
		if len(ins.ii.NetworkInterfaces) != len(wantNetworks[i]) {
			t.Errorf("Got %d nics, want %d", len(ins.ii.NetworkInterfaces), len(wantNetworks[i]))
		}
		for i, wantIPs := range wantNetworks[i] {
			nic := ins.ii.NetworkInterfaces[i]
			gotIPs := [4]string{nic.NetworkIP, nic.AccessConfigs[0].NatIP, nic.Ipv6Address}
			if len(nic.Ipv6AccessConfigs) > 0 {
				gotIPs[3] = nic.Ipv6AccessConfigs[0].ExternalIpv6
			}
			if gotIPs != wantIPs {
				t.Errorf("Got ips=%v, want=%v", gotIPs, wantIPs)
			}
		}
	}
}
