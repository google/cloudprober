// Copyright 2018 Google Inc.
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

package client

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/targets/rds/client/proto"
	pb "github.com/google/cloudprober/targets/rds/proto"
	"github.com/google/cloudprober/targets/rds/server"
	serverpb "github.com/google/cloudprober/targets/rds/server/proto"
)

type testProvider struct {
	resources []*pb.Resource
}

var testResourcesMap = map[string][]*pb.Resource{
	"test_provider1": {
		{
			Name: proto.String("testR21"),
			Ip:   proto.String("10.0.2.1"),
		},
		{
			Name: proto.String("testR22"),
			Ip:   proto.String("10.0.2.2"),
		},
	},
}

func (tp *testProvider) ListResources(*pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	return &pb.ListResourcesResponse{
		Resources: tp.resources,
	}, nil
}

func testServer(t *testing.T, testResourcesMap map[string][]*pb.Resource) *server.Server {
	testProviders := make(map[string]server.Provider)
	for tp, testResources := range testResourcesMap {
		testProviders[tp] = &testProvider{
			resources: testResources,
		}
	}
	srv, err := server.New(context.Background(), &serverpb.ServerConf{Addr: proto.String(":0")}, testProviders, &logger.Logger{})
	if err != nil {
		t.Fatalf("Got error starting RDS server: %v", err)
	}
	return srv
}

func TestListAndResolve(t *testing.T) {
	srv := testServer(t, testResourcesMap)
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go srv.Start(ctx, nil)

	for tp, testResources := range testResourcesMap {
		c := &configpb.ClientConf{
			ServerAddr: proto.String(srv.Addr().String()),
			Request: &pb.ListResourcesRequest{
				Provider: proto.String(tp),
			},
		}
		client, err := New(c, &logger.Logger{})
		if err != nil {
			t.Fatalf("Got error initializing RDS client: %v", err)
		}
		client.refreshState()

		// Test List()
		list := client.List()
		for i, res := range testResources {
			if res.GetName() != list[i] {
				t.Errorf("Didn't get expected resource. Got: %s, Want: %s", list[i], res.GetName())
			}
		}

		// Test Resolve()
		for _, res := range testResources {
			ip, err := client.Resolve(res.GetName(), 4)
			if err != nil {
				t.Errorf("Error resolving %s, err: %v", res.GetName(), err)
			}
			if ip.String() != res.GetIp() {
				t.Errorf("Didn't get expected IP for %s. Got: %s, Want: %s", res.GetName(), ip.String(), res.GetIp())
			}
		}
	}
}
