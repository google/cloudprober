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
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	pb "github.com/google/cloudprober/rds/proto"
)

func TestListMessages(t *testing.T) {
	lister := &pubsubMsgsLister{
		cache: make(map[string]map[string]time.Time),
	}
	lister.cache["s1"] = map[string]time.Time{
		"m1": time.Now().Add(-6 * time.Minute),
		"m2": time.Now().Add(-1 * time.Minute),
	}
	lister.cache["s2"] = map[string]time.Time{
		"m3": time.Now().Add(-1 * time.Minute),
	}

	// No filter
	want := []string{"m1", "m2", "m3"}
	resources, err := lister.listResources(nil)
	if err != nil {
		t.Errorf("Got error while listing resources: %v", err)
	}
	compareResources(t, resources, want)

	// Subscription s1, updated within 5m
	want = []string{"m2"}
	resources, err = lister.listResources(&pb.ListResourcesRequest{
		Filter: []*pb.Filter{
			&pb.Filter{
				Key:   proto.String("subscription"),
				Value: proto.String("s1"),
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

	// Subscription s1 and s2
	want = []string{"m1", "m2", "m3"}
	resources, _ = lister.listResources(&pb.ListResourcesRequest{
		Filter: []*pb.Filter{
			&pb.Filter{
				Key:   proto.String("subscription"),
				Value: proto.String("s"),
			},
		},
	})

	if err != nil {
		t.Errorf("Got error while listing resources: %v", err)
	}
	compareResources(t, resources, want)
}
