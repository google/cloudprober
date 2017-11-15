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
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/sysvars"
	"github.com/google/cloudprober/targets"
)

type serverConnStats struct {
	sync.Mutex
	msgCt map[string]int
}

func startUDPServer(ctx context.Context, t *testing.T, drop bool, delay time.Duration) (int, *serverConnStats) {
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		t.Fatalf("Starting UDP server failed: %v", err)
	}
	t.Logf("Recv addr: %s", conn.LocalAddr().String())
	// Simple loop to ECHO data.
	scs := &serverConnStats{
		msgCt: make(map[string]int),
	}

	go func() {
		timeout := time.Millisecond * 100
		maxLen := 1500
		b := make([]byte, maxLen)
		for {
			select {
			case <-ctx.Done():
				conn.Close()
				return
			default:
			}

			conn.SetReadDeadline(time.Now().Add(timeout))
			msgLen, addr, err := conn.ReadFromUDP(b)
			t.Logf("Message from %d %s %v", msgLen, addr.String(), err)
			if err != nil {
				if !isClientTimeout(err) {
					t.Logf("Error receiving message: %v", err)
				}
				continue
			}
			scs.Lock()
			scs.msgCt[addr.String()]++
			scs.Unlock()
			if drop {
				continue
			}
			if delay != 0 {
				time.Sleep(delay)
			}
			conn.SetWriteDeadline(time.Now().Add(timeout))
			if _, err := conn.WriteToUDP(b[:msgLen], addr); err != nil {
				t.Logf("Error sending message %s: %v", b[:msgLen], err)
			}
			t.Logf("Sent message to %s", addr.String())
		}
	}()

	return conn.LocalAddr().(*net.UDPAddr).Port, scs
}

func runProbe(ctx context.Context, t *testing.T, port int, interval, timeout time.Duration, pktsToSend int, scs *serverConnStats) *Probe {
	sysvars.Init(&logger.Logger{}, nil)
	p := &Probe{}
	opts := &options.Options{
		Targets:  targets.StaticTargets("localhost"),
		Interval: interval,
		Timeout:  timeout,
		ProbeConf: &ProbeConf{
			Port:       proto.Int32(int32(port)),
			NumTxPorts: proto.Int32(2),
			IpVersion:  proto.Int32(6),
		},
	}
	if err := p.Init("udp", opts); err != nil {
		t.Fatalf("Error initialzing UDP probe")
	}
	p.targets = p.opts.Targets.List()
	p.initProbeRunResults()
	for _, conn := range p.connList {
		go p.recvLoop(ctx, conn)
	}

	time.Sleep(time.Second)
	for i := 0; i < pktsToSend; i++ {
		p.runProbe()
		time.Sleep(time.Second)
	}

	scs.Lock()
	defer scs.Unlock()
	t.Logf("Echo server stats: %v", scs.msgCt)

	return p
}

func TestSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	port, scs := startUDPServer(ctx, t, false, 0)
	t.Logf("Will send to UDP port %d", port)
	var pktCount int64 = 2
	p := runProbe(ctx, t, port, time.Second*2, time.Second, int(pktCount), scs)

	scs.Lock()
	defer scs.Unlock()
	if len(scs.msgCt) != int(pktCount) {
		t.Errorf("Got packets over %d connections, required %d", len(scs.msgCt), pktCount)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	res := p.res["localhost"]
	if res.total.Int64() != pktCount || res.success.Int64() != pktCount {
		t.Errorf("Got total=%d, success=%d, want both to be %d", res.total.Int64(), res.success.Int64(), pktCount)
	}
}

func TestLossAndDelayed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pktCount int64 = 2
	cases := []struct {
		name     string
		drop     bool
		delay    time.Duration
		interval time.Duration
		delayCt  int64
	}{
		{"loss", true, 0, time.Second, 0},
		{"delayed", false, time.Second, time.Second / 2, pktCount},
	}

	for _, c := range cases {
		port, scs := startUDPServer(ctx, t, c.drop, c.delay)
		t.Logf("Case(%s): started server on port %d with loss %v delay %v", c.name, port, c.drop, c.delay)
		p := runProbe(ctx, t, port, time.Second*2, c.interval, int(pktCount), scs)

		p.mu.Lock()
		defer p.mu.Unlock()
		res := p.res["localhost"]
		if res.total.Int64() != pktCount {
			t.Errorf("Case(%s): got total=%d, want %d", c.name, res.total.Int64(), pktCount)
		}
		if res.success.Int64() != 0 {
			t.Errorf("Case(%s): got success=%d want 0", c.name, res.success.Int64())
		}
		if res.delayed.Int64() != c.delayCt {
			t.Errorf("Case(%s): got delayed=%d, want %d", c.name, res.delayed.Int64(), c.delayCt)
		}
	}
}
