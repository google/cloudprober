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

func testPodInfo(name, ns, ip string, labels map[string]string) *podInfo {
	pi := &podInfo{Metadata: kMetadata{Name: name, Namespace: ns, Labels: labels}}
	pi.Status.PodIP = ip
	return pi
}

func TestListResources(t *testing.T) {
	pl := &podsLister{}
	pl.names = []string{"podA", "podB", "podC"}
	pl.cache = map[string]*podInfo{
		"podA": testPodInfo("podA", "nsAB", "10.1.1.1", map[string]string{"app": "appA"}),
		"podB": testPodInfo("podB", "nsAB", "10.1.1.2", map[string]string{"app": "appB"}),
		"podC": testPodInfo("podC", "nsC", "10.1.1.3", map[string]string{"app": "appC", "func": "web"}),
	}

	tests := []struct {
		desc         string
		nameFilter   string
		filters      map[string]string
		labelsFilter map[string]string
		wantPods     []string
		wantErr      bool
	}{
		{
			desc:    "bad filter key, expect error",
			filters: map[string]string{"names": "pod(B|C)"},
			wantErr: true,
		},
		{
			desc:     "only name filter for podB and podC",
			filters:  map[string]string{"name": "pod(B|C)"},
			wantPods: []string{"podB", "podC"},
		},
		{
			desc:     "name and namespace filter for podB",
			filters:  map[string]string{"name": "pod(B|C)", "namespace": "nsAB"},
			wantPods: []string{"podB"},
		},
		{
			desc:     "only namespace filter for podA and podB",
			filters:  map[string]string{"namespace": "nsAB"},
			wantPods: []string{"podA", "podB"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var filtersPB []*pb.Filter
			for k, v := range test.filters {
				filtersPB = append(filtersPB, &pb.Filter{Key: proto.String(k), Value: proto.String(v)})
			}

			results, err := pl.listResources(filtersPB)
			if err != nil {
				if !test.wantErr {
					t.Errorf("got unexpected error: %v", err)
				}
				return
			}

			var gotNames []string
			for _, res := range results {
				gotNames = append(gotNames, res.GetName())
			}

			if !reflect.DeepEqual(gotNames, test.wantPods) {
				t.Errorf("pods.listResources: got=%v, expected=%v", gotNames, test.wantPods)
			}
		})
	}
}

func TestParseResourceList(t *testing.T) {
	podsListFile := "./testdata/pods.json"
	data, err := ioutil.ReadFile(podsListFile)

	if err != nil {
		t.Fatalf("error reading test data file: %s", podsListFile)
	}
	_, podsByName, err := parsePodsJSON(data)

	if err != nil {
		t.Fatalf("Error while parsing pods JSON data: %v", err)
	}

	cpPod := "cloudprober-54778d95f5-7hqtd"
	if podsByName[cpPod] == nil {
		t.Errorf("didn't get pod by the name: %s", cpPod)
	}

	// Verify that we got the pending pod.
	if podsByName["test"] != nil {
		t.Error("got a non-running pod in the list: test")
	}

	if podsByName[cpPod].Metadata.Labels["app"] != "cloudprober" {
		t.Errorf("cloudprober pod app label: got=%s, want=cloudprober", podsByName[cpPod].Metadata.Labels["app"])
	}

	cpPodIP := "10.28.0.3"
	if podsByName[cpPod].Status.PodIP != cpPodIP {
		t.Errorf("cloudprober pod ip: got=%s, want=%s", podsByName[cpPod].Status.PodIP, cpPodIP)
	}
}
