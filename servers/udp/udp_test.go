// Copyright 2017 Google Inc.
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

package udp

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/servers/udp/proto"
)

// Return true if the underlying error indicates a udp.Client timeout.
// In our case, we're using the ReadTimeout- time until response is read.
func isClientTimeout(err error) bool {
	e, ok := err.(*net.OpError)
	return ok && e != nil && e.Timeout()
}

func sendAndTestResponse(t *testing.T, c *configpb.ServerConf, conn net.Conn) {
	size := rand.Intn(1024)
	data := make([]byte, size)
	rand.Read(data)

	var err error
	m, err := conn.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if m < len(data) {
		t.Errorf("Wrote only %d of %d bytes", m, len(data))
	}

	switch c.GetType() {
	case configpb.ServerConf_ECHO:
		rcvd := make([]byte, size)
		n, err := conn.Read(rcvd)
		if err != nil {
			t.Fatal(err)
		}

		if m != n {
			t.Errorf("Sent %d bytes, got %d bytes", m, n)
		}
		if !bytes.Equal(data, rcvd) {
			t.Errorf("Data mismatch: Sent '%v', Got '%v'", data, rcvd)
		}
	case configpb.ServerConf_DISCARD:
		timeout := time.Duration(10) * time.Millisecond
		conn.SetReadDeadline(time.Now().Add(timeout))
		rcvd := make([]byte, size)
		n, err := conn.Read(rcvd)
		if err != nil {
			if isClientTimeout(err) {
				// Success, timed out with no response
				return
			}
			t.Fatal(err)
		}
		if n > 0 {
			t.Errorf("Received data (%v)! (Should be discarded)", rcvd)
		}
	}
}

func TestEchoServer(t *testing.T) {
	testConfig := &configpb.ServerConf{
		Port: proto.Int32(int32(0)),
		Type: configpb.ServerConf_ECHO.Enum(),
	}
	testServer(t, testConfig)
}

func TestDiscardServer(t *testing.T) {
	testConfig := &configpb.ServerConf{
		Port: proto.Int32(int32(0)),
		Type: configpb.ServerConf_DISCARD.Enum(),
	}
	testServer(t, testConfig)
}

func testServer(t *testing.T, testConfig *configpb.ServerConf) {
	l := &logger.Logger{}
	server, err := New(context.Background(), testConfig, l)
	if err != nil {
		t.Fatalf("Error creating a new server: %v", err)
	}
	serverAddr := fmt.Sprintf("localhost:%d", server.conn.LocalAddr().(*net.UDPAddr).Port)
	go server.Start(context.Background(), nil)
	// try 100 Samples
	for i := 0; i < 100; i++ {
		conn, err := net.Dial("udp", serverAddr)
		if err != nil {
			t.Fatal(err)
		}
		sendAndTestResponse(t, testConfig, conn)
		conn.Close()
	}
	// try 10 samples on the same connection
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for i := 0; i < 10; i++ {
		sendAndTestResponse(t, testConfig, conn)
	}
}
