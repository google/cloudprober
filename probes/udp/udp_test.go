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
	"os"
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
			if err != nil {
				if !isClientTimeout(err) {
					t.Logf("Error receiving message: %v", err)
				}
				continue
			}
			t.Logf("Message from %s, size: %d", addr.String(), msgLen)
			scs.Lock()
			scs.msgCt[addr.String()]++
			scs.Unlock()
			if drop {
				continue
			}
			go func(b []byte, addr *net.UDPAddr) {
				if delay != 0 {
					time.Sleep(delay)
				}
				conn.SetWriteDeadline(time.Now().Add(timeout))
				if _, err := conn.WriteToUDP(b, addr); err != nil {
					t.Logf("Error sending message %s: %v", b, err)
				}
				t.Logf("Sent message to %s", addr.String())
			}(append([]byte{}, b[:msgLen]...), addr)
		}
	}()

	return conn.LocalAddr().(*net.UDPAddr).Port, scs
}

func runProbe(ctx context.Context, t *testing.T, port int, interval, timeout time.Duration, pktsToSend int, scs *serverConnStats) *Probe {
	os.Setenv("DEBUG", "true")
	sysvars.Init(&logger.Logger{}, nil)
	p := &Probe{}
	ipVersion := int32(6)
	if _, ok := os.LookupEnv("TRAVIS"); ok {
		ipVersion = 4
	}
	opts := &options.Options{
		Targets:  targets.StaticTargets("localhost"),
		Interval: interval,
		Timeout:  timeout,
		ProbeConf: &ProbeConf{
			Port:       proto.Int32(int32(port)),
			NumTxPorts: proto.Int32(2),
			IpVersion:  proto.Int32(ipVersion),
		},
	}
	if err := p.Init("udp", opts); err != nil {
		t.Fatalf("Error initializing UDP probe: %v", err)
	}
	p.targets = p.opts.Targets.List()
	p.initProbeRunResults()

	for _, conn := range p.connList {
		go p.recvLoop(ctx, conn)
	}

	time.Sleep(time.Second)
	go func() {
		flushTicker := time.NewTicker(p.flushIntv)
		for {
			select {
			case <-ctx.Done():
				flushTicker.Stop()
				return
			case <-flushTicker.C:
				p.processPackets()
			}
		}
	}()

	time.Sleep(interval)
	for i := 0; i < pktsToSend; i++ {
		p.runProbe()
		time.Sleep(interval)
	}

	// Sleep for 2*statsExportIntv, to make sure that stats are updated and
	// exported.
	time.Sleep(2 * interval)
	time.Sleep(2 * timeout)

	scs.Lock()
	defer scs.Unlock()
	if len(scs.msgCt) != len(p.connList) {
		t.Errorf("Got packets over %d connections, required %d", len(scs.msgCt), p.connList)
	}
	t.Logf("Echo server stats: %v", scs.msgCt)

	return p
}

func TestSuccessMultipleCases(t *testing.T) {
	cases := []struct {
		name     string
		interval time.Duration
		timeout  time.Duration
		delay    time.Duration
		pktCount int64
	}{
		// 10 packets, at the interval of 200ms, with 200ms timeout and 20ms delay on server.
		{"success_normal", time.Second / 2, time.Second / 5, time.Second / 50, 5},
		// 20 packets, at the interval of 100ms, with 1000ms timeout and 50ms delay on server.
		{"success_timeout_larger_than_interval_1", time.Second / 10, time.Second, time.Second / 20, 20},
		// 20 packets, at the interval of 100ms, with 1000ms timeout and 200ms delay on server.
		{"success_timeout_larger_than_interval_2", time.Second / 10, time.Second, time.Second / 5, 20},
	}

	for _, c := range cases {
		ctx, cancelCtx := context.WithCancel(context.Background())
		port, scs := startUDPServer(ctx, t, false, c.delay)
		t.Logf("Case(%s): started server on port %d with delay %v", c.name, port, c.delay)
		p := runProbe(ctx, t, port, c.interval, c.timeout, int(c.pktCount), scs)
		cancelCtx()

		res := p.res["localhost"]
		if res.total != c.pktCount {
			t.Errorf("Case(%s): got total=%d, want %d", c.name, res.total, c.pktCount)
		}
		if res.success != c.pktCount {
			t.Errorf("Case(%s): got success=%d want %d", c.name, res.success, c.pktCount)
		}
		if res.delayed != 0 {
			t.Errorf("Case(%s): got delayed=%d, want 0", c.name, res.delayed)
		}
	}
}

func TestLossAndDelayed(t *testing.T) {
	var pktCount int64 = 10
	cases := []struct {
		name     string
		drop     bool
		interval time.Duration
		timeout  time.Duration
		delay    time.Duration
		delayCt  int64
	}{
		// 10 packets, at the interval of 100ms, with 50ms timeout and drop on server.
		{"loss", true, time.Second / 10, time.Second / 20, 0, 0},
		// 10 packets, at the interval of 100ms, with 50ms timeout and 67ms delay on server.
		{"delayed_1", false, time.Second / 10, time.Second / 20, time.Second / 15, pktCount},
		// 10 packets, at the interval of 100ms, with 250ms timeout and 300ms delay on server.
		{"delayed_2", false, time.Second / 10, time.Second / 4, 300*time.Millisecond, pktCount},
	}

	for _, c := range cases {
		ctx, cancelCtx := context.WithCancel(context.Background())
		port, scs := startUDPServer(ctx, t, c.drop, c.delay)
		t.Logf("Case(%s): started server on port %d with loss %v delay %v", c.name, port, c.drop, c.delay)
		p := runProbe(ctx, t, port, c.interval, c.timeout, int(pktCount), scs)
		cancelCtx()

		res := p.res["localhost"]
		if res.total != pktCount {
			t.Errorf("Case(%s): got total=%d, want %d", c.name, res.total, pktCount)
		}
		if res.success != 0 {
			t.Errorf("Case(%s): got success=%d want 0", c.name, res.success)
		}
		if res.delayed != c.delayCt {
			t.Errorf("Case(%s): got delayed=%d, want %d", c.name, res.delayed, c.delayCt)
		}
	}
}
