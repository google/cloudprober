// Copyright 2018-2019 The Cloudprober Authors.
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
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/client/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server"
	serverpb "github.com/google/cloudprober/rds/server/proto"
	dnsRes "github.com/google/cloudprober/targets/resolver"
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
		{
			Name:   proto.String("testR22v6"),
			Ip:     proto.String("::1"),
			Port:   proto.Int32(8080),
			Labels: map[string]string{"zone": "us-central1-a"},
		},
		{
			Name:   proto.String("testR3"),
			Ip:     proto.String("testR3.test.com"),
			Port:   proto.Int32(80),
			Labels: map[string]string{"zone": "us-central1-c"},
		},
		// Duplicate resource, it should be removed from the result.
		{
			Name:   proto.String("testR3"),
			Ip:     proto.String("testR3.test.com"),
			Port:   proto.Int32(80),
			Labels: map[string]string{"zone": "us-central1-c"},
		},
	},
}

var expectedIPByVersion = map[string]map[int]string{
	"testR21": map[int]string{
		0: "10.0.2.1",
		4: "10.0.2.1",
		6: "err",
	},
	"testR22": map[int]string{
		0: "10.0.2.2",
		4: "10.0.2.2",
		6: "err",
	},
	"testR22v6": map[int]string{
		0: "::1",
		4: "err",
		6: "::1",
	},
	"testR3": map[int]string{
		0: "10.1.1.2",
		4: "10.1.1.2",
		6: "::2",
	},
}

var testNameToIP = map[string][]net.IP{
	"testR3.test.com": []net.IP{net.ParseIP("10.1.1.2"), net.ParseIP("::2")},
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
		client.resolver = dnsRes.NewWithResolve(func(name string) ([]net.IP, error) {
			return testNameToIP[name], nil
		})

		client.refreshState(time.Second)

		// Test ListEndpoint(), note that we remove the duplicate resource from the
		// exepected output.
		epList := client.ListEndpoints()
		expectedList := testResources[:len(testResources)-1]
		for i, res := range expectedList {
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
		for _, res := range expectedList {
			for _, ipVer := range []int{0, 4, 6} {
				t.Run(fmt.Sprintf("resolve_%s_for_IPv%d", res.GetName(), ipVer), func(t *testing.T) {
					expectedIP := expectedIPByVersion[res.GetName()][ipVer]

					var gotIP string
					ip, err := client.Resolve(res.GetName(), ipVer)
					if err != nil {
						t.Logf("Error resolving %s, err: %v", res.GetName(), err)
						gotIP = "err"
					} else {
						gotIP = ip.String()
					}

					if gotIP != expectedIP {
						t.Errorf("Didn't get expected IP for %s. Got: %s, Want: %s", res.GetName(), gotIP, expectedIP)
					}
				})
			}
		}
	}
}
