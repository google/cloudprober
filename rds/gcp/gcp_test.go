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

package gcp

import (
	"reflect"
	"testing"

	pb "github.com/google/cloudprober/rds/proto"
	serverconfigpb "github.com/google/cloudprober/rds/server/proto"
)

func testGCPConfig(t *testing.T, pc *serverconfigpb.Provider, projects []string, gceInstances bool, rtcConfig, pubsubTopic, apiVersion string, reEvalSec int) {
	t.Helper()

	if pc.GetId() != DefaultProviderID {
		t.Errorf("pc.GetId()=%s, wanted=%s", pc.GetId(), DefaultProviderID)
	}
	c := pc.GetGcpConfig()

	if !reflect.DeepEqual(c.GetProject(), projects) {
		t.Errorf("Projects in GCP config=%v, wanted=%v", c.GetProject(), projects)
	}

	if c.GetApiVersion() != apiVersion {
		t.Errorf("API verion in GCP config=%v, wanted=%v", c.GetApiVersion(), apiVersion)
	}

	if !gceInstances {
		if c.GetGceInstances() != nil {
			t.Errorf("c.GetGceInstances()=%v, wanted=nil", c.GetGceInstances())
		}
	} else {
		if c.GetGceInstances() == nil {
			t.Fatal("c.GetGceInstances() is nil, wanted=not-nil")
		}
		if c.GetGceInstances().GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("GCE instance reEvalSec=%d, wanted=%d", c.GetGceInstances().GetReEvalSec(), reEvalSec)
		}
	}

	// Verify that RTC config is set correctly.
	if rtcConfig == "" {
		if c.GetRtcVariables() != nil {
			t.Errorf("c.GetRtcVariables()=%v, wanted=nil", c.GetRtcVariables())
		}
	} else {
		if c.GetRtcVariables() == nil {
			t.Fatalf("c.GetRtcVariables()=nil, wanted=not-nil")
		}
		if c.GetRtcVariables().GetRtcConfig()[0].GetName() != rtcConfig {
			t.Errorf("RTC config=%s, wanted=%s", c.GetRtcVariables().GetRtcConfig()[0].GetName(), rtcConfig)
		}
		if c.GetRtcVariables().GetRtcConfig()[0].GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("RTC config reEvalSec=%d, wanted=%d", c.GetRtcVariables().GetRtcConfig()[0].GetReEvalSec(), reEvalSec)
		}
	}

	// Verify that Pub/Sub topic is set correctly.
	if pubsubTopic == "" {
		if c.GetPubsubMessages() != nil {
			t.Errorf("c.GetPubsubMessages()=%v, wanted=nil", c.GetPubsubMessages())
		}
	} else {
		if c.GetPubsubMessages() == nil {
			t.Fatalf("c.GetRtcVariables()=nil, wanted=not-nil")
		}
		if c.GetPubsubMessages().GetSubscription()[0].GetTopicName() != pubsubTopic {
			t.Errorf("Pubsub topic name=%s, wanted=%s", c.GetPubsubMessages().GetSubscription()[0].GetTopicName(), pubsubTopic)
		}
	}
}

func TestDefaultProviderConfig(t *testing.T) {
	projects := []string{"p1", "p2"}
	resTypes := map[string]string{
		ResourceTypes.GCEInstances: "",
	}
	apiVersion := ""
	c := DefaultProviderConfig(projects, resTypes, 10, apiVersion)
	testGCPConfig(t, c, projects, true, "", "", apiVersion, 10)

	// RTC and pub-sub
	testRTCConfig := "rtc-config"
	testPubsubTopic := "pubsub-topic"
	apiVersion = "v1"
	resTypes = map[string]string{
		ResourceTypes.RTCVariables:   testRTCConfig,
		ResourceTypes.PubsubMessages: testPubsubTopic,
	}
	c = DefaultProviderConfig(projects, resTypes, 10, apiVersion)
	testGCPConfig(t, c, projects, false, testRTCConfig, testPubsubTopic, apiVersion, 10)

	// GCE instances, RTC and pub-sub
	resTypes = map[string]string{
		ResourceTypes.GCEInstances:   "",
		ResourceTypes.RTCVariables:   testRTCConfig,
		ResourceTypes.PubsubMessages: testPubsubTopic,
	}
	c = DefaultProviderConfig(projects, resTypes, 10, apiVersion)
	testGCPConfig(t, c, projects, true, testRTCConfig, testPubsubTopic, apiVersion, 10)
}

type dummyLister struct {
	name string
}

func (dl *dummyLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	return []*pb.Resource{}, nil
}

func TestListersForResourcePath(t *testing.T) {
	projects := []string{"p1", "p2"}
	p := &Provider{
		projects: projects,
		listers:  make(map[string]map[string]lister),
	}

	for _, project := range projects {
		p.listers[project] = map[string]lister{
			ResourceTypes.GCEInstances: &dummyLister{
				name: project,
			},
		}
	}

	testCases := []struct {
		rp, listerName string
		wantErr        bool
	}{
		{"gce_instances", projects[0], false},
		{"gce_instances/", projects[0], false},
		{"gce_instances/p1", "p1", false},
		{"gce_instances/p2", "p2", false},
		{"gce_instances/p3", "", true}, // unknown project
		{"instances/p3", "", true},     // unknown resource_type
	}

	for _, tc := range testCases {
		t.Run("testing_for_resource_path:"+tc.rp, func(t *testing.T) {
			lr, err := p.listerForResourcePath(tc.rp)

			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tc.wantErr)
			}

			if err != nil {
				return
			}

			dlr := lr.(*dummyLister)
			if dlr.name != tc.listerName {
				t.Errorf("got lister=%s, wanted=%s", dlr.name, tc.listerName)
			}
		})
	}

}
