package server

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/google/cloudprober/rds/proto"
)

type testProvider struct {
	resources []*pb.Resource
}

func (tp *testProvider) ListResources(*pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	return &pb.ListResourcesResponse{
		Resources: tp.resources,
	}, nil
}

func TestListResources(t *testing.T) {
	testResources := []*pb.Resource{
		&pb.Resource{
			Name: proto.String("testR1"),
			Ip:   proto.String("IP1"),
		},
		&pb.Resource{
			Name: proto.String("testR2"),
			Ip:   proto.String("IP2"),
		},
	}
	srv := &Server{
		providers: map[string]Provider{
			"test_provider": &testProvider{
				resources: testResources,
			},
		},
	}
	res, err := srv.ListResources(context.Background(), &pb.ListResourcesRequest{
		Provider:     proto.String("test_provider"),
		ResourcePath: proto.String("rp"),
	})
	if err != nil {
		t.Errorf("Got unexpected error while listing test resources: %v", err)
	}
	if !reflect.DeepEqual(res.Resources, testResources) {
		t.Errorf("Didn't get expected resource. Got=%v, Want=%v", res.Resources, testResources)
	}
}
