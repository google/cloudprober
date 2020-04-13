// Copyright 2018-2019 Google Inc.
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
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/client/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server"
	serverpb "github.com/google/cloudprober/rds/server/proto"
)

type testProvider struct {
	resources []*pb.Resource
}

var testResourcesMap = map[string][]*pb.Resource{
	"test_provider1": {
		{
			Name:   proto.String("testR21"),
			Ip:     proto.String("10.0.2.1"),
			Port:   proto.Int32(80),
			Labels: map[string]string{"zone": "us-central1-b"},
		},
		{
			Name:   proto.String("testR22"),
			Ip:     proto.String("10.0.2.2"),
			Port:   proto.Int32(8080),
			Labels: map[string]string{"zone": "us-central1-a"},
		},
	},
}

func (tp *testProvider) ListResources(*pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	return &pb.ListResourcesResponse{
		Resources: tp.resources,
	}, nil
}

func setupTestServer(ctx context.Context, t *testing.T, testResourcesMap map[string][]*pb.Resource) *server.Server {
	t.Helper()

	testProviders := make(map[string]server.Provider)
	for tp, testResources := range testResourcesMap {
		testProviders[tp] = &testProvider{
			resources: testResources,
		}
	}

	srv, err := server.New(context.Background(), &serverpb.ServerConf{}, testProviders, &logger.Logger{})
	if err != nil {
		t.Fatalf("Got error creating RDS server: %v", err)
	}

	return srv
}

func TestListAndResolve(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	srv := setupTestServer(ctx, t, testResourcesMap)

	for tp, testResources := range testResourcesMap {
		c := &configpb.ClientConf{
			Request: &pb.ListResourcesRequest{
				Provider: proto.String(tp),
			},
		}
		client, err := New(c, srv.ListResources, &logger.Logger{})
		if err != nil {
			t.Fatalf("Got error initializing RDS client: %v", err)
		}
		client.refreshState(time.Second)

		// Test ListEndpoint()
		epList := client.ListEndpoints()
		for i, res := range testResources {
			if epList[i].Name != res.GetName() {
				t.Errorf("Resource name: got=%s, want=%s", epList[i].Name, res.GetName())
			}

			if epList[i].Port != int(res.GetPort()) {
				t.Errorf("Resource port: got=%d, want=%d", epList[i].Port, res.GetPort())
			}

			if !reflect.DeepEqual(epList[i].Labels, res.GetLabels()) {
				t.Errorf("Resource labels: got=%v, want=%v", epList[i].Labels, res.GetLabels())
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
