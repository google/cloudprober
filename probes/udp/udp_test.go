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
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	configpb "github.com/google/cloudprober/probes/udp/proto"
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

const numTxPorts = 2

func runProbe(ctx context.Context, t *testing.T, interval, timeout time.Duration, probesToSend int, scs *serverConnStats, conf configpb.ProbeConf) *Probe {
	sysvars.Init(&logger.Logger{}, nil)
	p := &Probe{}
	ipVersion := int32(6)
	if _, ok := os.LookupEnv("TRAVIS"); ok {
		ipVersion = 4
	}
	conf.NumTxPorts = proto.Int32(numTxPorts)
	conf.IpVersion = proto.Int32(ipVersion)
	opts := &options.Options{
		Targets:   targets.StaticTargets("localhost"),
		Interval:  interval,
		Timeout:   timeout,
		ProbeConf: &conf,
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
	for i := 0; i < probesToSend; i++ {
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
		t.Errorf("Got packets over %d connections, required %d", len(scs.msgCt), len(p.connList))
	}
	t.Logf("Echo server stats: %v", scs.msgCt)

	return p
}

func TestSuccessMultipleCasesResultPerPort(t *testing.T) {
	cases := []struct {
		name        string
		interval    time.Duration
		timeout     time.Duration
		delay       time.Duration
		probeCount  int
		useAllPorts bool
		pktCount    int64
	}{
		// 10 probes, probing each target from 2 ports, at the interval of 200ms, with 100ms timeout and 10ms delay on server.
		{"success_normal", 200, 100, 10, 10, true, 10},
		// 20 probes, probing each target from 2 ports, at the interval of 100ms, with 1000ms timeout and 50ms delay on server.
		{"success_timeout_larger_than_interval_1", 100, 1000, 50, 20, true, 20},
		// 20 probes, probing each target from 2 ports, at the interval of 100ms, with 1000ms timeout and 200ms delay on server.
		{"success_timeout_larger_than_interval_2", 100, 1000, 200, 20, true, 20},
		// 10 probes, probing each target just once, at the interval of 200ms, with 100ms timeout and 10ms delay on server.
		{"single_port", 200, 100, 10, 10, false, 5},
	}

	for _, c := range cases {
		ctx, cancelCtx := context.WithCancel(context.Background())
		port, scs := startUDPServer(ctx, t, false, c.delay*time.Millisecond)
		t.Logf("Case(%s): started server on port %d with delay %v", c.name, port, c.delay)
		conf := configpb.ProbeConf{
			UseAllTxPortsPerProbe: proto.Bool(c.useAllPorts),
			Port:                proto.Int32(int32(port)),
			ExportMetricsByPort: proto.Bool(true)}
		p := runProbe(ctx, t, c.interval*time.Millisecond, c.timeout*time.Millisecond, c.probeCount, scs, conf)
		cancelCtx()

		if len(p.connList) != numTxPorts {
			t.Errorf("Case(%s): len(p.connList)=%d, want %d", c.name, len(p.connList), numTxPorts)
		}
		for _, port := range p.srcPortList {
			res := p.res[flow{port, "localhost"}]
			if res.total != c.pktCount {
				t.Errorf("Case(%s): p.res[_].total=%d, want %d", c.name, res.total, c.pktCount)
			}
			if res.success != c.pktCount {
				t.Errorf("Case(%s): p.res[_].success=%d want %d", c.name, res.success, c.pktCount)
			}
			if res.delayed != 0 {
				t.Errorf("Case(%s): p.res[_].delayed=%d, want 0", c.name, res.delayed)
			}
		}
	}
}

func TestSuccessMultipleCasesDefaultResult(t *testing.T) {
	cases := []struct {
		name        string
		interval    time.Duration
		timeout     time.Duration
		delay       time.Duration
		probeCount  int
		useAllPorts bool
		pktCount    int64
	}{
		// 10 probes, probing each target from 2 ports, at the interval of 200ms, with 100ms timeout and 10ms delay on server.
		{"success_normal", 200, 100, 10, 10, true, 20},
		// 20 probes, probing each target from 2 ports, at the interval of 100ms, with 1000ms timeout and 50ms delay on server.
		{"success_timeout_larger_than_interval_1", 100, 1000, 50, 20, true, 40},
		// 20 probes, probing each target from 2 ports, at the interval of 100ms, with 1000ms timeout and 200ms delay on server.
		{"success_timeout_larger_than_interval_2", 100, 1000, 200, 20, true, 40},
		// 10 probes, probing each target just once, at the interval of 200ms, with 100ms timeout and 10ms delay on server.
		{"single_port", 200, 100, 10, 10, false, 10},
	}

	for _, c := range cases {
		ctx, cancelCtx := context.WithCancel(context.Background())
		port, scs := startUDPServer(ctx, t, false, c.delay*time.Millisecond)
		t.Logf("Case(%s): started server on port %d with delay %v", c.name, port, c.delay)
		conf := configpb.ProbeConf{
			UseAllTxPortsPerProbe: proto.Bool(c.useAllPorts),
			Port:                proto.Int32(int32(port)),
			ExportMetricsByPort: proto.Bool(false)}
		p := runProbe(ctx, t, c.interval*time.Millisecond, c.timeout*time.Millisecond, c.probeCount, scs, conf)
		cancelCtx()

		if len(p.connList) != numTxPorts {
			t.Errorf("Case(%s): len(p.connList)=%d, want %d", c.name, len(p.connList), numTxPorts)
		}
		res := p.res[flow{"", "localhost"}]
		if res.total != c.pktCount {
			t.Errorf("Case(%s): p.res[_].total=%d, want %d", c.name, res.total, c.pktCount)
		}
		if res.success != c.pktCount {
			t.Errorf("Case(%s): p.res[_].success=%d want %d", c.name, res.success, c.pktCount)
		}
		if res.delayed != 0 {
			t.Errorf("Case(%s): p.res[_].delayed=%d, want 0", c.name, res.delayed)
		}
	}
}

func extractMetric(em *metrics.EventMetrics, key string) int64 {
	return em.Metric(key).(*metrics.Int).Int64()
}

func TestExport(t *testing.T) {
	res := probeResult{
		total:   3,
		success: 2,
		delayed: 1,
		latency: metrics.NewFloat(100.),
	}
	conf := configpb.ProbeConf{
		ExportMetricsByPort: proto.Bool(true),
		Port:                proto.Int32(1234),
	}
	m := res.EventMetrics("probe", flow{"port", "target"}, &conf)
	if r := extractMetric(m, "total-per-port"); r != 3 {
		t.Errorf("extractMetric(m,\"total-per-port\")=%d, want 3", r)
	}
	if r := extractMetric(m, "success-per-port"); r != 2 {
		t.Errorf("extractMetric(m,\"success-per-port\")=%d, want 2", r)
	}
	if got, want := m.Label("src_port"), "port"; got != want {
		t.Errorf("m.Label(\"src_port\")=%q, want %q", got, want)
	}
	if got, want := m.Label("dst_port"), "1234"; got != want {
		t.Errorf("m.Label(\"dst_port\")=%q, want %q", got, want)
	}
	conf = configpb.ProbeConf{
		ExportMetricsByPort: proto.Bool(false),
		Port:                proto.Int32(1234),
	}
	m = res.EventMetrics("probe", flow{"port", "target"}, &conf)
	if r := extractMetric(m, "total"); r != 3 {
		t.Errorf("extractMetric(m,\"total\")=%d, want 3", r)
	}
	if r := extractMetric(m, "success"); r != 2 {
		t.Errorf("extractMetric(m,\"success\")=%d, want 2", r)
	}
	if got, want := m.Label("src_port"), ""; got != want {
		t.Errorf("m.Label(\"src_port\")=%q, want %q", got, want)
	}
	if got, want := m.Label("dst_port"), ""; got != want {
		t.Errorf("m.Label(\"dst_port\")=%q, want %q", got, want)
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
		{"loss", true, 100, 50, 0, 0},
		// 10 packets, at the interval of 100ms, with 50ms timeout and 67ms delay on server.
		{"delayed_1", false, 100, 50, 67, pktCount},
		// 10 packets, at the interval of 100ms, with 250ms timeout and 300ms delay on server.
		{"delayed_2", false, 100, 250, 300, pktCount},
	}

	for _, c := range cases {
		ctx, cancelCtx := context.WithCancel(context.Background())
		port, scs := startUDPServer(ctx, t, c.drop, c.delay*time.Millisecond)
		t.Logf("Case(%s): started server on port %d with loss %v delay %v", c.name, port, c.drop, c.delay)
		conf := configpb.ProbeConf{
			UseAllTxPortsPerProbe: proto.Bool(true),
			Port:                proto.Int32(int32(port)),
			ExportMetricsByPort: proto.Bool(true)}
		p := runProbe(ctx, t, c.interval*time.Millisecond, c.timeout*time.Millisecond, int(pktCount), scs, conf)
		cancelCtx()

		if len(p.connList) != numTxPorts {
			t.Errorf("Case(%s): len(p.connList)=%d, want %d", c.name, len(p.connList), numTxPorts)
		}
		for _, port := range p.srcPortList {
			res := p.res[flow{port, "localhost"}]
			if res.total != pktCount {
				t.Errorf("Case(%s): p.res[_].total=%d, want %d", c.name, res.total, pktCount)
			}
			if res.success != 0 {
				t.Errorf("Case(%s): p.res[_].success=%d want 0", c.name, res.success)
			}
			if res.delayed != c.delayCt {
				t.Errorf("Case(%s): p.res[_].delayed=%d, want %d", c.name, res.delayed, c.delayCt)
			}
		}
	}
}
