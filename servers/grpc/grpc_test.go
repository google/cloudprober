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

// Unit tests for grpc server.
package grpc

import (
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/servers/grpc/proto"
	grpcpb "github.com/google/cloudprober/servers/grpc/proto"
	spb "github.com/google/cloudprober/servers/grpc/proto"
	"google.golang.org/grpc"
	"google3/go/context/context"
)

func TestGRPCSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &configpb.ServerConf{
		Port: proto.Int32(0),
	}
	l := &logger.Logger{}

	srv, err := New(ctx, cfg, l)
	if err != nil {
		t.Fatalf("Unable to create grpc server: %v", err)
	}
	go srv.Start(ctx, nil)

	listenAddr := srv.ln.Addr().String()
	conn, err := grpc.Dial(listenAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Unable to connect to grpc server at %v: %v", listenAddr, err)
	}

	client := grpcpb.NewProberClient(conn)
	timedCtx, timedCancel := context.WithTimeout(ctx, time.Second)
	defer timedCancel()
	sReq := &spb.StatusRequest{}
	sResp, err := client.ServerStatus(timedCtx, sReq)
	if err != nil {
		t.Errorf("ServerStatus call error: %v", err)
	}
	t.Logf("Uptime: %v", sResp.GetUptimeUs())
	if sResp.GetUptimeUs() == 0 {
		t.Error("Uptime not being incremented.")
	}

	timedCtx, timedCancel = context.WithTimeout(ctx, time.Second)
	defer timedCancel()
	msg := []byte("test message")
	echoReq := &spb.EchoMessage{Blob: msg}
	echoResp, err := client.Echo(timedCtx, echoReq)
	if err != nil {
		t.Errorf("Echo call error: %v", err)
	}
	t.Logf("EchoResponse: <%v>", string(echoResp.Blob))
	if !reflect.DeepEqual(echoResp.Blob, echoReq.Blob) {
		t.Errorf("Echo response mismatch: got %v want %v", echoResp.Blob, echoReq.Blob)
	}
}
