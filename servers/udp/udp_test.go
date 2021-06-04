// Copyright 2017 The Cloudprober Authors.
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
	"sync"
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

func sendAndTestResponse(t *testing.T, c *configpb.ServerConf, conn net.Conn) error {
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

	timeout := time.Duration(10) * time.Millisecond
	conn.SetReadDeadline(time.Now().Add(timeout))

	switch c.GetType() {
	case configpb.ServerConf_ECHO:
		rcvd := make([]byte, size)
		n, err := conn.Read(rcvd)
		if err != nil {
			t.Error(err)
			return err
		}

		if m != n {
			t.Errorf("Sent %d bytes, got %d bytes", m, n)
		}
		if !bytes.Equal(data, rcvd) {
			t.Errorf("Data mismatch: Sent '%v', Got '%v'", data, rcvd)
		}
	case configpb.ServerConf_DISCARD:
		rcvd := make([]byte, size)
		n, err := conn.Read(rcvd)
		if err != nil {
			if isClientTimeout(err) {
				// Success, timed out with no response
				return nil
			}
			t.Error(err)
			return err
		}
		if n > 0 {
			t.Errorf("Received data (%v)! (Should be discarded)", rcvd)
		}
	}
	return nil
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
	for i := 0; i < 10; i++ {
		t.Logf("Creating connection %d to %s", i, serverAddr)
		conn, err := net.Dial("udp", serverAddr)
		if err != nil {
			t.Fatal(err)
		}
		if err = sendAndTestResponse(t, testConfig, conn); err != nil {
			conn.Close()
		}
		server.mu.RLock()
		t.Logf("Rcvd: %d, Sent: %d", server.rcvd, server.sent)
		server.mu.RUnlock()
	}
	// try 10 samples on the same connection
	t.Logf("Creating many-packet connection to %s", serverAddr)
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for i := 0; i < 10; i++ {
		if err := sendAndTestResponse(t, testConfig, conn); err != nil {
			return
		}
	}
}

func TestServerStop(t *testing.T) {
	t.Run("ECHO mode", func(t *testing.T) {
		testServerStopWithConfig(t, &configpb.ServerConf{
			Port: proto.Int32(int32(0)),
			Type: configpb.ServerConf_ECHO.Enum(),
		})
	})
	t.Run("Discard mode", func(t *testing.T) {
		testServerStopWithConfig(t, &configpb.ServerConf{
			Port: proto.Int32(int32(0)),
			Type: configpb.ServerConf_DISCARD.Enum(),
		})
	})
}

func testServerStopWithConfig(t *testing.T, testConfig *configpb.ServerConf) {
	t.Helper()

	server, err := New(context.Background(), testConfig, &logger.Logger{})
	if err != nil {
		t.Fatalf("Error creating a new server: %v", err)
	}
	serverAddr := fmt.Sprintf("localhost:%d", server.conn.LocalAddr().(*net.UDPAddr).Port)

	var wg sync.WaitGroup
	ctx, cancelF := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		server.Start(ctx, nil)
		wg.Done()
	}()

	go func() {
		time.Sleep(1 * time.Second)
		cancelF()
	}()

	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		t.Errorf("Error connecting to test UDP server (%s): %v", serverAddr, err)
	}
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	for i := 0; true; i++ {
		_, err := conn.Write(make([]byte, 10))
		if err == nil {
			continue
		}
		t.Logf("Stopped writing packet due to error: %v, sent %d packets", err, i+1)
		break
	}

	wg.Wait()
}
