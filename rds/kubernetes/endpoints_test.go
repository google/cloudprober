package kubernetes

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/google/cloudprober/rds/server/filter"
)

func TestParseEndpoints(t *testing.T) {
	epListFile := "./testdata/endpoints.json"
	data, err := ioutil.ReadFile(epListFile)

	if err != nil {
		t.Fatalf("error reading test data file: %s", epListFile)
	}
	_, epByKey, err := parseEndpointsJSON(data)
	if err != nil {
		t.Fatalf("error reading test data file: %s", epListFile)
	}

	testKeys := []resourceKey{
		{"default", "cloudprober"},
		{"default", "cloudprober-test"},
		{"system", "kubernetes"},
	}
	for _, key := range testKeys {
		if epByKey[key] == nil {
			t.Errorf("didn't get endpoints for %+v", key)
		}
	}

	for _, key := range testKeys[:1] {
		epi := epByKey[key]
		if epi.Metadata.Labels["app"] != "cloudprober" {
			t.Errorf("cloudprober endpoints app label: got=%s, want=cloudprober", epi.Metadata.Labels["app"])
		}

		if len(epi.Subsets) != 1 {
			t.Errorf("cloudprober endpoints subsets count: got=%d, want=1", len(epi.Subsets))
		}

		eps := epi.Subsets[0]
		var ips, pods, nodes []string
		for _, addr := range eps.Addresses {
			ips = append(ips, addr.IP)
			nodes = append(nodes, addr.NodeName)
			if addr.TargetRef.Kind != "Pod" {
				t.Errorf("Unexpected target ref kind for addr (%s): %s", addr.IP, addr.TargetRef.Kind)
			}
			pods = append(pods, addr.TargetRef.Name)
		}

		expectedIPs := []string{"10.28.0.3", "10.28.2.3", "10.28.2.6"}
		if !reflect.DeepEqual(ips, expectedIPs) {
			t.Errorf("cloudprober endpoints addresses: got=%v, want=%v", ips, expectedIPs)
		}

		expectedNodes := []string{"gke-cluster-1-default-pool-abd8ad35-ccr7", "gke-cluster-1-default-pool-abd8ad35-mzh9", "gke-cluster-1-default-pool-abd8ad35-mzh9"}
		if !reflect.DeepEqual(nodes, expectedNodes) {
			t.Errorf("cloudprober endpoints nodes: got=%v, want=%v", nodes, expectedNodes)
		}

		expectedPods := []string{"cloudprober-test-577cf7bbcc-c7l5p", "cloudprober-test-577cf7bbcc-qnrvg", "cloudprober-54778d95f5-vms2d"}
		if !reflect.DeepEqual(pods, expectedPods) {
			t.Errorf("cloudprober endpoints pods: got=%v, want=%v", pods, expectedPods)
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
			IP        string
			NodeName  string
			TargetRef struct {
				Kind string
				Name string
			}
		}{
			{
				IP:       ips[0],
				NodeName: "n1",
				TargetRef: struct {
					Kind, Name string
				}{
					Kind: "Pod",
					Name: "test-pod",
				},
			},
			{
				IP:       ips[1],
				NodeName: "n2",
			},
		},
		Ports: []struct {
			Name string
			Port int
		}{
			{Port: 9313},
			{Name: "rds", Port: 9314},
			{Name: "not-rds", Port: 9315}, // should be excluded
		},
	}

	portFilter, err := filter.NewRegexFilter("^(rds|9313)$")
	if err != nil {
		t.Fatal(err)
	}
	resources := epi.resources(portFilter, nil)

	// We'll get 4 resources = 2 ports x 2 IPs
	if len(resources) != 4 {
		t.Errorf("cloudprober resources len(resources)=%d, want=4", len(resources))
	}

	var names, resIPs []string
	var labels []map[string]string
	var ports []int32

	for _, res := range resources {
		t.Logf("name=%s, IP=%s", res.GetName(), res.GetIp())
		names = append(names, res.GetName())
		labels = append(labels, res.GetLabels())
		resIPs = append(resIPs, res.GetIp())
		ports = append(ports, res.GetPort())
	}

	expectedNames := []string{"cloudprober_10.0.0.1_9313", "cloudprober_10.0.0.2_9313", "cloudprober_10.0.0.1_rds", "cloudprober_10.0.0.2_rds"}
	if !reflect.DeepEqual(names, expectedNames) {
		t.Errorf("Cloudprober endpoints resource names=%v, want=%v", names, expectedNames)
	}

	expectedLabels := []map[string]string{
		{"app": "lCloudprober", "node": "n1", "pod": "test-pod"},
		{"app": "lCloudprober", "node": "n2"},
		{"app": "lCloudprober", "node": "n1", "pod": "test-pod"},
		{"app": "lCloudprober", "node": "n2"},
	}
	if !reflect.DeepEqual(labels, expectedLabels) {
		t.Errorf("Cloudprober endpoints resource labels=%v, want=%v", labels, expectedLabels)
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
