package kubernetes

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func TestParseEndpoints(t *testing.T) {
	epListFile := "./testdata/endpoints.json"
	data, err := ioutil.ReadFile(epListFile)

	if err != nil {
		t.Fatalf("error reading test data file: %s", epListFile)
	}
	_, epByName, err := parseEndpointsJSON(data)
	if err != nil {
		t.Fatalf("error reading test data file: %s", epListFile)
	}

	testNames := []string{"cloudprober", "cloudprober-test", "kubernetes"}
	for _, testP := range testNames {
		if epByName[testP] == nil {
			t.Errorf("didn't get endpoints by the name: %s", testP)
		}
	}

	for _, name := range testNames[:1] {
		epi := epByName[name]
		if epi.Metadata.Labels["app"] != "cloudprober" {
			t.Errorf("cloudprober endpoints app label: got=%s, want=cloudprober", epi.Metadata.Labels["app"])
		}

		if len(epi.Subsets) != 1 {
			t.Errorf("cloudprober endpoints subsets count: got=%d, want=1", len(epi.Subsets))
		}

		eps := epi.Subsets[0]
		var ips []string
		for _, addr := range eps.Addresses {
			ips = append(ips, addr.IP)
		}
		expectedIPs := []string{"10.28.0.3", "10.28.2.3", "10.28.2.6"}
		if !reflect.DeepEqual(ips, expectedIPs) {
			t.Errorf("cloudprober endpoints addresses: got=%v, want=%v", ips, expectedIPs)
		}

		if len(eps.Ports) != 1 {
			t.Errorf("cloudprober endpoints len(eps.Ports)=%d, want=1", len(eps.Ports))
		}

		if eps.Ports[0].Port != 9313 {
			t.Errorf("cloudprober endpoints eps.Ports[0].Port=%d, want=9313", eps.Ports[0].Port)
		}
	}
}

// TestEndpointsToResources tests endpoints to RDS resources conversion.
func TestEndpointsToResources(t *testing.T) {
	epName := "cloudprober"
	appLabel := "lCloudprober"
	ips := []string{"10.0.0.1", "10.0.0.2"}

	epi := &epInfo{
		Metadata: kMetadata{
			Name:   epName,
			Labels: map[string]string{"app": appLabel},
		},
		Subsets: make([]epSubset, 1),
	}

	epi.Subsets[0] = epSubset{
		Addresses: []struct {
			IP string
		}{
			{IP: ips[0]},
			{IP: ips[1]},
		},
		Ports: []struct {
			Name string
			Port int
		}{
			{Port: 9313},
			{Name: "rds", Port: 9314},
		},
	}

	resources := epi.resources()

	// We'll get 4 resources = 2 ports x 2 IPs
	if len(resources) != 4 {
		t.Errorf("cloudprober resources len(resources)=%d, want=4", len(resources))
	}

	var names, resIPs []string
	var ports []int32
	for _, res := range resources {
		t.Logf("name=%s, IP=%s", res.GetName(), res.GetIp())
		names = append(names, res.GetName())
		resIPs = append(resIPs, res.GetIp())
		ports = append(ports, res.GetPort())
	}

	expectedNames := []string{"cloudprober_10.0.0.1_9313", "cloudprober_10.0.0.2_9313", "cloudprober_10.0.0.1_rds", "cloudprober_10.0.0.2_rds"}
	if !reflect.DeepEqual(names, expectedNames) {
		t.Errorf("Cloudprober endpoints resource names=%v, want=%v", names, expectedNames)
	}

	expectedIPs := []string{"10.0.0.1", "10.0.0.2", "10.0.0.1", "10.0.0.2"}
	if !reflect.DeepEqual(resIPs, expectedIPs) {
		t.Errorf("Cloudprober endpoints resource IPs=%v, want=%v", resIPs, expectedIPs)
	}

	expectedPorts := []int32{9313, 9313, 9314, 9314}
	if !reflect.DeepEqual(ports, expectedPorts) {
		t.Errorf("Cloudprober endpoints resource Ports=%v, want=%v", ports, expectedPorts)
	}
}
