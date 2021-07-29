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
	"github.com/google/cloudprober/targets/endpoint"
	dnsRes "github.com/google/cloudprober/targets/resolver"
)

type testProvider struct {
	resources           []*pb.Resource
	supportCacheControl bool
	lastModified        int64
	requestCache        []*pb.ListResourcesRequest
	responseCache       []*pb.ListResourcesResponse
}

func (tp *testProvider) verifyRequestResponse(t *testing.T, runCount int, ifModifiedSince, lastModified int64) {
	t.Helper()

	if len(tp.requestCache) != runCount {
		t.Errorf("Unexpected requests cache length: %d, want: %d", len(tp.requestCache), runCount)
		return
	}
	if len(tp.responseCache) != runCount {
		t.Errorf("Unexpected responses cache length: %d, want: %d", len(tp.responseCache), runCount)
		return
	}
	if tp.requestCache[runCount-1].GetIfModifiedSince() != ifModifiedSince {
		t.Errorf("Request's if_modified_since: %d, want: %d", tp.requestCache[runCount-1].GetIfModifiedSince(), ifModifiedSince)
	}
	if tp.responseCache[runCount-1].GetLastModified() != lastModified {
		t.Errorf("Response's last_modified: %d, want: %d", tp.responseCache[runCount-1].GetLastModified(), lastModified)
	}
}

var testProviderName = "test_provider1"

var testResources = []*pb.Resource{
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
}

var testResourcesMap = map[string][]*pb.Resource{
	testProviderName: testResources,
}

// Expected resource list, note that we remove the duplicate resource from the
// expected output.
var expectedList = testResources[:len(testResources)-1]

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

func (tp *testProvider) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	tp.requestCache = append(tp.requestCache, req)
	var resp *pb.ListResourcesResponse
	defer func() {
		tp.responseCache = append(tp.responseCache, resp)
	}()

	// If provider doesn't support cache control, just return all resources.
	if !tp.supportCacheControl {
		resp = &pb.ListResourcesResponse{
			Resources: tp.resources,
		}
		return resp, nil
	}

	// If we support cache-control, we should at least add last-modified to the
	// response.
	resp = &pb.ListResourcesResponse{
		LastModified: proto.Int64(tp.lastModified),
	}

	// If request contains if_modified_since, use that field to decide if we
	// should add resources to the response.
	if req.GetIfModifiedSince() != 0 {
		if tp.lastModified <= req.GetIfModifiedSince() {
			return resp, nil
		}
	}
	resp.Resources = tp.resources

	return resp, nil
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

func verifyEndpoints(t *testing.T, epList []endpoint.Endpoint, expectedList []*pb.Resource) {
	t.Helper()

	if len(epList) != len(expectedList) {
		t.Fatalf("Got endpoints:\n%v\nExpected:\n%v", epList, expectedList)
		return
	}

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
}

func TestListAndResolve(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	srv := setupTestServer(ctx, t, map[string][]*pb.Resource{
		testProviderName: testResources,
	})

	c := &configpb.ClientConf{
		Request: &pb.ListResourcesRequest{
			Provider: proto.String(testProviderName),
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
	verifyEndpoints(t, client.ListEndpoints(), expectedList)

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

func TestCacheBehaviorWithServerSupport(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	initLastModified := time.Now().Unix()
	tp := &testProvider{
		resources:           testResources,
		supportCacheControl: true,
		lastModified:        initLastModified,
	}

	srv, err := server.New(ctx, &serverpb.ServerConf{}, map[string]server.Provider{testProviderName: tp}, &logger.Logger{})
	if err != nil {
		t.Fatalf("Got error creating RDS server: %v", err)
	}

	c := &configpb.ClientConf{
		Request: &pb.ListResourcesRequest{
			Provider: proto.String(testProviderName),
		},
	}
	client, err := New(c, srv.ListResources, &logger.Logger{})
	if err != nil {
		t.Fatalf("Got error initializing RDS client: %v", err)
	}
	client.resolver = dnsRes.NewWithResolve(func(name string) ([]net.IP, error) {
		return testNameToIP[name], nil
	})

	// Since New calls refreshState, there should already be a request.
	runCount := 1
	tp.verifyRequestResponse(t, runCount, 0, initLastModified)
	if client.lastModified != initLastModified {
		t.Errorf("Client's last modified: %d, want: %d.", client.lastModified, initLastModified)
	}
	verifyEndpoints(t, client.ListEndpoints(), expectedList)

	// Verify that refreshing client state doesn't change its last-modified and
	// we still list all the resources.
	tp.resources = testResources[1:] // This should have no effect
	client.refreshState(time.Second)
	runCount++
	tp.verifyRequestResponse(t, runCount, initLastModified, initLastModified)
	if client.lastModified != initLastModified {
		t.Errorf("Client's last modified: %d, expected: %d.", client.lastModified, initLastModified)
	}
	verifyEndpoints(t, client.ListEndpoints(), expectedList)

	// Change provider's last modified to trigger client's refresh.
	newLastModified := initLastModified + 1
	tp.lastModified = newLastModified
	tp.resources = testResources[1:]
	client.refreshState(time.Second)
	runCount++
	tp.verifyRequestResponse(t, runCount, initLastModified, newLastModified)
	if client.lastModified == initLastModified || client.lastModified != newLastModified {
		t.Errorf("Unexpected last modified timestamps. Previous: %v, Now: %v, Expected: %v", initLastModified, client.lastModified, newLastModified)
	}
	// State will get updated this time, hence shorter expectedList
	verifyEndpoints(t, client.ListEndpoints(), expectedList[1:])
}

func TestCacheBehaviorWithoutServerSupport(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	tp := &testProvider{
		resources:           testResources,
		supportCacheControl: false,
	}
	srv, err := server.New(ctx, &serverpb.ServerConf{}, map[string]server.Provider{testProviderName: tp}, &logger.Logger{})
	if err != nil {
		t.Fatalf("Got error creating RDS server: %v", err)
	}

	c := &configpb.ClientConf{
		Request: &pb.ListResourcesRequest{
			Provider: proto.String(testProviderName),
		},
	}
	client, err := New(c, srv.ListResources, &logger.Logger{})
	if err != nil {
		t.Fatalf("Got error initializing RDS client: %v", err)
	}
	client.resolver = dnsRes.NewWithResolve(func(name string) ([]net.IP, error) {
		return testNameToIP[name], nil
	})

	// Since New calls refreshState, there should already be a request.
	runCount := 1
	tp.verifyRequestResponse(t, runCount, 0, 0)
	if client.lastModified != 0 {
		t.Errorf("Client's last modified: %d, want: %d.", client.lastModified, 0)
	}
	verifyEndpoints(t, client.ListEndpoints(), expectedList)

	// Verify that refreshing client's state changes its state.
	tp.resources = testResources[1:]
	client.refreshState(time.Second)
	runCount++
	tp.verifyRequestResponse(t, runCount, 0, 0)
	verifyEndpoints(t, client.ListEndpoints(), expectedList[1:])
}
