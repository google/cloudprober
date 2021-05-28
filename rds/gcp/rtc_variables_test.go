// Copyright 2018 The Cloudprober Authors.
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
	"sort"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	pb "github.com/google/cloudprober/rds/proto"
	runtimeconfig "google.golang.org/api/runtimeconfig/v1beta1"
)

type testVar struct {
	name       string
	updateTime time.Time
}

func (tv *testVar) rtcVar() *runtimeconfig.Variable {
	return &runtimeconfig.Variable{
		Name:       tv.name,
		UpdateTime: tv.updateTime.Format(time.RFC3339Nano),
	}
}

func TestProcessVar(t *testing.T) {

	// Valid variable
	v1UpdateTime := time.Now().Add(-6 * time.Minute)
	testVar := &runtimeconfig.Variable{
		Name:       "config1/v1",
		UpdateTime: v1UpdateTime.Format(time.RFC3339Nano),
	}
	got, err := processVar(testVar)
	if err != nil {
		t.Errorf("processVar: got error while processing the testVar (%v), err: %v", testVar, err)
	}
	if got.name != "v1" {
		t.Errorf("processVar: got: %s, want: v1", got.name)
	}
	if !got.updateTime.Equal(v1UpdateTime) {
		t.Errorf("processVars: got var[%s].updateTime: %s, want var[v1]: updateTime: %v", got.name, got.updateTime, v1UpdateTime)
	}

	// Invalid variable
	testVar = &runtimeconfig.Variable{
		Name:       "invalidname",
		UpdateTime: v1UpdateTime.Format(time.RFC3339Nano),
	}
	got, err = processVar(testVar)
	if err == nil {
		t.Errorf("processVar: didn't get error for the invalid testVar: %v", testVar)
	}

}

func compareResources(t *testing.T, resources []*pb.Resource, want []string) {
	t.Helper()

	var got []string
	for _, res := range resources {
		got = append(got, res.GetName())
	}

	sort.Strings(got)
	sort.Strings(want)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("RTC listResources: got=%v, want=%v", got, want)
	}
}

func TestListRTCVars(t *testing.T) {
	rvl := &rtcVariablesLister{
		cache: make(map[string][]*rtcVar),
	}
	rvl.cache["c1"] = []*rtcVar{&rtcVar{"v1", time.Now().Add(-6 * time.Minute)}, &rtcVar{"v2", time.Now().Add(-1 * time.Minute)}}
	rvl.cache["c2"] = []*rtcVar{&rtcVar{"v3", time.Now().Add(-1 * time.Minute)}}

	// No filter
	want := []string{"v1", "v2", "v3"}
	resources, err := rvl.listResources(nil)
	if err != nil {
		t.Errorf("Got error while listing resources: %v", err)
	}
	compareResources(t, resources, want)

	// Config c1, updated within 5m
	want = []string{"v2"}
	resources, err = rvl.listResources(&pb.ListResourcesRequest{
		Filter: []*pb.Filter{
			&pb.Filter{
				Key:   proto.String("config_name"),
				Value: proto.String("c1"),
			},
			&pb.Filter{
				Key:   proto.String("updated_within"),
				Value: proto.String("5m"),
			},
		},
	})

	if err != nil {
		t.Errorf("Got error while listing resources: %v", err)
	}
	compareResources(t, resources, want)

	// Config c1 and c2
	want = []string{"v1", "v2", "v3"}
	resources, _ = rvl.listResources(&pb.ListResourcesRequest{
		Filter: []*pb.Filter{
			&pb.Filter{
				Key:   proto.String("config_name"),
				Value: proto.String("c"),
			},
		},
	})

	if err != nil {
		t.Errorf("Got error while listing resources: %v", err)
	}
	compareResources(t, resources, want)
}
