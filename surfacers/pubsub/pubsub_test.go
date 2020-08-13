// Copyright 2020 Google Inc.
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

package pubsub

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/surfacers/common/compress"
	configpb "github.com/google/cloudprober/surfacers/pubsub/proto"
	"google.golang.org/api/option"
	pb "google.golang.org/genproto/googleapis/pubsub/v1"
	pb_grpc "google.golang.org/genproto/googleapis/pubsub/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// testServer is the underlying service implementor. It is not intended to be used
// directly.
type testServer struct {
	pb_grpc.PublisherServer
	pb_grpc.SubscriberServer

	topics map[string]*pb.Topic
	msgs   []*Message // all messages ever published
	wg     sync.WaitGroup
	nextID int
}

// A Message is a message that was published to the server.
type Message struct {
	Data       []byte
	Attributes map[string]string
}

func (s *testServer) CreateTopic(_ context.Context, t *pb.Topic) (*pb.Topic, error) {
	if s.topics[t.Name] != nil {
		return nil, status.Errorf(codes.AlreadyExists, "topic %q", t.Name)
	}
	s.topics[t.Name] = t
	return t, nil
}

func (s *testServer) GetTopic(_ context.Context, req *pb.GetTopicRequest) (*pb.Topic, error) {
	if t := s.topics[req.Topic]; t != nil {
		return t, nil
	}
	return nil, status.Errorf(codes.NotFound, "topic %q", req.Topic)
}

func (s *testServer) Publish(_ context.Context, req *pb.PublishRequest) (*pb.PublishResponse, error) {
	var ids []string
	for _, pm := range req.Messages {
		m := &Message{
			Data:       pm.Data,
			Attributes: pm.Attributes,
		}
		ids = append(ids, fmt.Sprintf("m%d", s.nextID))
		s.nextID++
		s.msgs = append(s.msgs, m)
	}
	return &pb.PublishResponse{MessageIds: ids}, nil
}

func TestSurfacer(t *testing.T) {
	for _, compression := range []bool{false, true} {
		t.Run(fmt.Sprintf("with_compression=%v", compression), func(t *testing.T) {
			l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 0))
			if err != nil {
				t.Fatalf("Error creating listener: %v")
			}
			defer l.Close()

			gSrv := grpc.NewServer()
			srv := &testServer{
				topics: map[string]*pb.Topic{},
			}

			pb_grpc.RegisterPublisherServer(gSrv, srv)
			pb_grpc.RegisterSubscriberServer(gSrv, srv)

			go func() {
				if err := gSrv.Serve(l); err != nil {
					t.Fatalf("gRPC server start: %v", err)
				}
			}()
			defer gSrv.Stop()

			// Connect to the server without using TLS.
			conn, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
			if err != nil {
				t.Fatalf("Error establishing connection to the test pubsub server (%s): %v", l.Addr().String(), err)
			}
			defer conn.Close()

			newPubsubClient = func(ctx context.Context, project string) (*pubsub.Client, error) {
				return pubsub.NewClient(ctx, project, option.WithGRPCConn(conn))
			}

			createSurfacerAndVerify(t, srv, compression)
		})
	}
}

func createSurfacerAndVerify(t *testing.T, srv *testServer, compression bool) {
	t.Helper()

	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	s, err := New(context.Background(), &configpb.SurfacerConf{
		TopicName:          proto.String("test-topic"),
		CompressionEnabled: proto.Bool(compression),
	}, &logger.Logger{})
	if err != nil {
		t.Fatalf("Error while creating new surfacer: %v", err)
	}

	testEM := []*metrics.EventMetrics{
		metrics.NewEventMetrics(time.Now()).AddMetric("float-test", metrics.NewInt(123456)),
		metrics.NewEventMetrics(time.Now()).AddMetric("float-test", metrics.NewInt(123457)),
	}

	var expectedMsgs []string
	for _, em := range testEM {
		s.Write(context.Background(), em)
		expectedMsgs = append(expectedMsgs, em.String())
	}

	// Closing the surfacer waits for inputs to be processed.
	s.close()

	srv.wg.Wait()

	if compression {
		b, err := compress.Compress([]byte(strings.Join(expectedMsgs, "\n") + "\n"))
		if err != nil {
			t.Fatalf("Error while compressing data (%s): %v", string(b), err)
		}
		expectedMsgs = []string{string(b)}
	}

	if len(srv.msgs) != len(expectedMsgs) {
		t.Fatalf("Got %d message, expected: %d", len(srv.msgs), len(expectedMsgs))
	}

	expectedAttributes := map[string]string{
		"starttime":  s.starttime,
		"compressed": map[bool]string{false: "false", true: "true"}[compression],
	}

	for i, msg := range srv.msgs {
		if !reflect.DeepEqual(msg.Attributes, expectedAttributes) {
			t.Errorf("Message attributes: %v, expected: %v", msg.Attributes, expectedAttributes)
		}
		if string(msg.Data) != expectedMsgs[i] {
			t.Errorf("Message data=%s, expected=%s", string(msg.Data), expectedMsgs[i])
		}
	}
}
