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
// TODO(manugarg): Add more tests after a bit of refactoring.

package lameduck

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	rdspb "github.com/google/cloudprober/rds/proto"
)

type mockLDLister struct {
	list []string
}

func (mldl *mockLDLister) List() []string {
	return mldl.list
}

func TestDefaultLister(t *testing.T) {
	list1 := []string{"list1"}

	// Initialize default lister with the given lister
	InitDefaultLister(nil, &mockLDLister{list1}, nil)
	lister, err := GetDefaultLister()
	if err != nil {
		t.Fatal(err)
	}
	gotList := lister.List()
	if !reflect.DeepEqual(gotList, list1) {
		t.Errorf("Default lister retured: %v, expected: %v", gotList, list1)
	}

	list2 := []string{"list2"}
	// Initialize default lister with the given lister. This time it should have
	// no impact as default lister is already initialized.
	InitDefaultLister(nil, &mockLDLister{list2}, nil)
	lister, err = GetDefaultLister()
	if err != nil {
		t.Fatal(err)
	}
	gotList = lister.List()
	if !reflect.DeepEqual(gotList, list1) {
		t.Errorf("Default lister retured: %v, expected: %v", gotList, list1)
	}
}

func TestList(t *testing.T) {
	li := lister{
		listResourcesFunc: funcListResources,
		project:           "test-project",
		rtcConfig:         "lame-duck-targets",
		pubsubTopic:       "lame-duck-targets",
	}
	if err := li.initClients(); err != nil {
		t.Errorf("lister.initClients(): %v", err)
	}

	names := li.List()
	wantNames := []string{"v1", "v2", "m1", "m2"}

	if !reflect.DeepEqual(names, wantNames) {
		t.Errorf("lister.List(): got=%v, want=%v", names, wantNames)
	}
}

// We use funcListResources to create RDS clients for testing purpose.
func funcListResources(ctx context.Context, in *rdspb.ListResourcesRequest) (*rdspb.ListResourcesResponse, error) {
	path := in.GetResourcePath()
	resType := strings.SplitN(path, "/", 2)[0]
	var resources []*rdspb.Resource

	switch resType {
	case "rtc_variables":
		resources = []*rdspb.Resource{
			{
				Name: proto.String("v1"),
			},
			{
				Name: proto.String("v2"),
			},
		}
	case "pubsub_messages":
		resources = []*rdspb.Resource{
			{
				Name: proto.String("m1"),
			},
			{
				Name: proto.String("m2"),
			},
		}
	default:
		return nil, errors.New("unsupported resource_type")
	}

	return &rdspb.ListResourcesResponse{Resources: resources}, nil
}
