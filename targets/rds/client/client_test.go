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
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/targets/rds/client/proto"
	pb "github.com/google/cloudprober/targets/rds/proto"
	"github.com/google/cloudprober/targets/rds/server"
	serverpb "github.com/google/cloudprober/targets/rds/server/proto"
	"google.golang.org/grpc"
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

func setupTestServer(ctx context.Context, t *testing.T, testResourcesMap map[string][]*pb.Resource) net.Addr {
	testProviders := make(map[string]server.Provider)
	for tp, testResources := range testResourcesMap {
		testProviders[tp] = &testProvider{
			resources: testResources,
		}
	}

	grpcServer := grpc.NewServer()
	_, err := server.New(context.Background(), &serverpb.ServerConf{}, testProviders, grpcServer, &logger.Logger{})
	if err != nil {
		t.Fatalf("Got error creating RDS server: %v", err)
	}
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Got error creating listener for RDS server: %v", err)
	}
	go grpcServer.Serve(ln)
	return ln.Addr()
}

func TestListAndResolve(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	addr := setupTestServer(ctx, t, testResourcesMap)

	for tp, testResources := range testResourcesMap {
		c := &configpb.ClientConf{
			ServerAddr: proto.String(addr.String()),
			Request: &pb.ListResourcesRequest{
				Provider: proto.String(tp),
			},
		}
		client, err := New(c, &logger.Logger{})
		if err != nil {
			t.Fatalf("Got error initializing RDS client: %v", err)
		}
		client.refreshState(time.Second)

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
