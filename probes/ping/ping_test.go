// Copyright 2017-2018 Google Inc.
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

package ping

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/options"
	configpb "github.com/google/cloudprober/probes/ping/proto"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/targets"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func peerToIP(peer net.Addr) string {
	switch peer := peer.(type) {
	case *net.UDPAddr:
		return peer.IP.String()
	case *net.IPAddr:
		return peer.IP.String()
	}
	return ""
}

// replyPkt creates an ECHO reply packet from the ECHO request packet.
func replyPkt(pkt []byte, ipVersion int) []byte {
	protocol := protocolICMP
	var typ icmp.Type
	typ = ipv4.ICMPTypeEchoReply
	if ipVersion == 6 {
		protocol = protocolIPv6ICMP
		typ = ipv6.ICMPTypeEchoReply
	}
	m, _ := icmp.ParseMessage(protocol, pkt)
	m.Type = typ
	b, _ := m.Marshal(nil)
	return b
}

// testICMPConn implements the icmpConn interface.
// It implements the following packets pipeline:
//      write(packet) --> sentPackets channel -> read() -> packet
// It has a per-target channel that receives packets through the "write" call.
// "read" call fetches packets from that channel and returns them to the
// caller.
type testICMPConn struct {
	sentPackets map[string](chan []byte)
	c           *configpb.ProbeConf

	flipLastByte   bool
	flipLastByteMu sync.Mutex
}

func newTestICMPConn(c *configpb.ProbeConf, targets []string) *testICMPConn {
	tic := &testICMPConn{
		c:           c,
		sentPackets: make(map[string](chan []byte)),
	}
	for _, target := range targets {
		tic.sentPackets[target] = make(chan []byte, c.GetPacketsPerProbe())
	}
	return tic
}

func (tic *testICMPConn) setFlipLastByte() {
	tic.flipLastByteMu.Lock()
	defer tic.flipLastByteMu.Unlock()
	tic.flipLastByte = true
}

func (tic *testICMPConn) read(buf []byte) (int, net.Addr, error) {
	// We create per-target select cases, with each target's select-case
	// pointing to that target's sentPackets channel.
	var cases []reflect.SelectCase
	var targets []string
	for t, ch := range tic.sentPackets {
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)})
		targets = append(targets, t)
	}

	// Select over the select cases.
	chosen, value, ok := reflect.Select(cases)
	if !ok {
		return 0, nil, fmt.Errorf("nothing to read")
	}

	pkt := value.Bytes()

	// Since we are echoing the packets, copy the received packet into the
	// provided buffer.
	respPkt := replyPkt(pkt, int(tic.c.GetIpVersion()))
	tic.flipLastByteMu.Lock()
	if tic.flipLastByte {
		lastByte := ^respPkt[len(respPkt)-1]
		respPkt = append(respPkt[:len(respPkt)-1], lastByte)
	}
	tic.flipLastByteMu.Unlock()

	copy(buf[0:len(pkt)], respPkt)
	peerIP, _ := resolveAddr(targets[chosen], int(tic.c.GetIpVersion()))

	var peer net.Addr
	peer = &net.IPAddr{IP: peerIP}
	if tic.c.GetUseDatagramSocket() {
		peer = &net.UDPAddr{IP: peerIP}
	}
	return len(pkt), peer, nil
}

// write simply queues packets into the sentPackets channel. These packets are
// retrieved by the "read" call.
func (tic *testICMPConn) write(b []byte, peer net.Addr) (int, error) {
	target := peerToIP(peer)
	tic.sentPackets[target] <- b
	return len(b), nil
}

func (tic *testICMPConn) setReadDeadline(deadline time.Time) {
}

func (tic *testICMPConn) close() {
}

// Sends packets and verifies
func sendAndCheckPackets(p *Probe, t *testing.T) {
	tic := newTestICMPConn(p.c, p.targets)
	p.conn = tic
	trackerChan := make(chan bool, int(p.c.GetPacketsPerProbe())*len(p.targets))
	p.resolveTargets()
	runID := p.newRunID()
	p.sendPackets(runID, trackerChan)

	protocol := protocolICMP
	var expectedMsgType icmp.Type
	expectedMsgType = ipv4.ICMPTypeEcho
	if p.c.GetIpVersion() == 6 {
		protocol = protocolIPv6ICMP
		expectedMsgType = ipv6.ICMPTypeEchoRequest
	}

	for _, target := range p.targets {
		if int(p.sent[target]) != int(p.c.GetPacketsPerProbe()) {
			t.Errorf("Mismatch in number of packets recorded to be sent. Sent: %d, Recorded: %d", p.c.GetPacketsPerProbe(), p.sent[target])
		}
		if len(tic.sentPackets[target]) != int(p.c.GetPacketsPerProbe()) {
			t.Errorf("Mismatch in number of packets received. Sent: %d, Got: %d", p.c.GetPacketsPerProbe(), len(tic.sentPackets[target]))
		}
		close(tic.sentPackets[target])
		for b := range tic.sentPackets[target] {
			// Make sure packets parse ok
			m, err := icmp.ParseMessage(protocol, b)
			if err != nil {
				t.Errorf("%v", err)
			}
			// Check packet type
			if m.Type != expectedMsgType {
				t.Errorf("Wrong packet type. Got: %v, expected: %v", m.Type, expectedMsgType)
			}
			// Check packet size
			if len(b) != int(p.c.GetPayloadSize())+8 {
				t.Errorf("Wrong packet size. Got: %d, expected: %d", len(b), int(p.c.GetPayloadSize())+8)
			}
			// Verify ICMP id and sequence number
			pkt, ok := m.Body.(*icmp.Echo)
			if !ok {
				t.Errorf("Wrong ICMP packet body")
			}
			if pkt.ID != int(runID) {
				t.Errorf("Got wrong ICMP ID. Got: %d, Expected: %d", pkt.ID, runID)
			}
			if pkt.Seq&0xff00 != int(runID)&0xff00 {
				t.Errorf("Got wrong ICMP base seq number. Got: %d, Expected: %d", pkt.Seq&0xff00, runID&0xff00)
			}
		}
	}
}

func newProbe(c *configpb.ProbeConf, t []string) (*Probe, error) {
	p := &Probe{
		name: "ping_test",
		c:    c,
		opts: &options.Options{
			Targets:  targets.StaticTargets(strings.Join(t, ",")),
			Interval: 2 * time.Second,
			Timeout:  time.Second,
		},
		ipVer:             int(c.GetIpVersion()),
		l:                 &logger.Logger{},
		sent:              make(map[string]int64),
		received:          make(map[string]int64),
		latency:           make(map[string]time.Duration),
		validationFailure: make(map[string]*metrics.Map),

		ip2target:   make(map[string]string),
		target2addr: make(map[string]net.Addr),
	}

	p.targets = p.opts.Targets.List()

	if err := p.configureIntegrityCheck(); err != nil {
		return nil, err
	}
	return p, p.setSourceFromConfig()
}

type intf struct {
	addrs []net.Addr
}

func (i *intf) Addrs() ([]net.Addr, error) {
	return i.addrs, nil
}

func mockInterfaceByName(iname string, addrs []string) {
	ips := make([]net.Addr, len(addrs))
	for i, a := range addrs {
		ips[i] = &net.IPAddr{IP: net.ParseIP(a)}
	}
	i := &intf{addrs: ips}
	probeutils.InterfaceByName = func(name string) (probeutils.Addr, error) {
		if name != iname {
			return nil, errors.New("device not found")
		}
		return i, nil
	}
}

func TestInitSourceIP(t *testing.T) {
	rows := []struct {
		name       string
		sourceIP   string
		sourceIntf string
		intf       string
		intfAddrs  []string
		want       string
		wantError  bool
	}{
		{
			name:     "Use ip if set",
			sourceIP: "1.1.1.1",
			want:     "1.1.1.1",
		},
		{
			name:       "Interface with no adders fails",
			sourceIntf: "eth1",
			intf:       "eth1",
			wantError:  true,
		},
		{
			name:       "Unknown interface fails",
			sourceIntf: "eth1",
			intf:       "eth0",
			wantError:  true,
		},
		{
			name:       "Uses first addr for interface",
			sourceIntf: "eth1",
			intf:       "eth1",
			intfAddrs:  []string{"1.1.1.1", "2.2.2.2"},
			want:       "1.1.1.1",
		},
	}

	for _, r := range rows {
		c := &configpb.ProbeConf{}
		if r.sourceIP != "" {
			c.Source = &configpb.ProbeConf_SourceIp{r.sourceIP}
		} else {
			c.Source = &configpb.ProbeConf_SourceInterface{r.sourceIntf}
			mockInterfaceByName(r.intf, r.intfAddrs)
		}
		p, err := newProbe(c, []string{})
		if (err != nil) != r.wantError {
			t.Errorf("Row %q: newProbe() gave error %q, want error is %v", r.name, err, r.wantError)
			continue
		}
		if r.wantError {
			continue
		}
		if p.source != r.want {
			t.Errorf("Row %q: p.source = %q, want %q", r.name, p.source, r.want)
		}
	}
}

// Test sendPackets IPv4, raw sockets
func TestSendPackets(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.Source = &configpb.ProbeConf_SourceIp{"1.1.1.1"}
	p, err := newProbe(c, []string{"2.2.2.2", "3.3.3.3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	sendAndCheckPackets(p, t)
}

// Test sendPackets IPv6, raw sockets
func TestSendPacketsIPv6(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.IpVersion = proto.Int32(6)
	c.Source = &configpb.ProbeConf_SourceIp{"::1"}
	p, err := newProbe(c, []string{"::2", "::3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	sendAndCheckPackets(p, t)
}

// Test sendPackets IPv6, raw sockets, no packets should come on IPv4 target
func TestSendPacketsIPv6ToIPv4Hosts(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.IpVersion = proto.Int32(6)
	c.Source = &configpb.ProbeConf_SourceIp{"::1"}
	p, err := newProbe(c, []string{"2.2.2.2"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	tic := newTestICMPConn(c, p.targets)
	p.conn = tic
	trackerChan := make(chan bool, int(c.GetPacketsPerProbe())*len(p.targets))
	p.resolveTargets()
	p.sendPackets(p.newRunID(), trackerChan)
	for _, target := range p.targets {
		if len(tic.sentPackets[target]) != 0 {
			t.Errorf("IPv6 probe: should not have received any packets for IPv4 only targets, but got %d packets", len(tic.sentPackets[target]))
		}
	}
}

// Test sendPackets IPv4, datagram sockets
func TestSendPacketsDatagramSocket(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.UseDatagramSocket = proto.Bool(true)
	c.Source = &configpb.ProbeConf_SourceIp{"1.1.1.1"}
	p, err := newProbe(c, []string{"2.2.2.2", "3.3.3.3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	sendAndCheckPackets(p, t)
}

// Test sendPackets IPv6, datagram sockets
func TestSendPacketsIPv6DatagramSocket(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.UseDatagramSocket = proto.Bool(true)
	c.IpVersion = proto.Int32(6)
	c.Source = &configpb.ProbeConf_SourceIp{"::1"}
	p, err := newProbe(c, []string{"::2", "::3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	sendAndCheckPackets(p, t)
}

// Test runProbe IPv4, raw sockets
func TestRunProbe(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.Source = &configpb.ProbeConf_SourceIp{"1.1.1.1"}
	p, err := newProbe(c, []string{"2.2.2.2", "3.3.3.3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	p.conn = newTestICMPConn(c, p.targets)
	p.runProbe()
	for _, target := range p.targets {
		glog.Infof("target: %s, sent: %d, received: %d, total_rtt: %s", target, p.sent[target], p.received[target], p.latency[target])
		if p.sent[target] == 0 || (p.sent[target] != p.received[target]) {
			t.Errorf("We are leaking packets. Sent: %d, Received: %d", p.sent[target], p.received[target])
		}
	}
}

// Test runProbe IPv6, raw sockets
func TestRunProbeIPv6(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.IpVersion = proto.Int32(6)
	c.Source = &configpb.ProbeConf_SourceIp{"::1"}
	p, err := newProbe(c, []string{"::2", "::3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	p.conn = newTestICMPConn(c, p.targets)
	p.runProbe()
	for _, target := range p.targets {
		glog.Infof("target: %s, sent: %d, received: %d, total_rtt: %s", target, p.sent[target], p.received[target], p.latency[target])
		if p.sent[target] == 0 || (p.sent[target] != p.received[target]) {
			t.Errorf("We are leaking packets. Sent: %d, Received: %d", p.sent[target], p.received[target])
		}
	}
}

// Test runProbe IPv4, datagram sockets
func TestRunProbeDatagram(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.UseDatagramSocket = proto.Bool(true)
	c.Source = &configpb.ProbeConf_SourceIp{"1.1.1.1"}
	p, err := newProbe(c, []string{"2.2.2.2", "3.3.3.3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	p.conn = newTestICMPConn(c, p.targets)
	p.runProbe()
	for _, target := range p.targets {
		glog.Infof("target: %s, sent: %d, received: %d, total_rtt: %s", target, p.sent[target], p.received[target], p.latency[target])
		if p.sent[target] == 0 || (p.sent[target] != p.received[target]) {
			t.Errorf("We are leaking packets. Sent: %d, Received: %d", p.sent[target], p.received[target])
		}
	}
}

// Test runProbe IPv6, datagram sockets
func TestRunProbeIPv6Datagram(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.UseDatagramSocket = proto.Bool(true)
	c.IpVersion = proto.Int32(6)
	c.Source = &configpb.ProbeConf_SourceIp{"::1"}
	p, err := newProbe(c, []string{"::2", "::3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	p.conn = newTestICMPConn(c, p.targets)
	p.runProbe()
	for _, target := range p.targets {
		glog.Infof("target: %s, sent: %d, received: %d, total_rtt: %s", target, p.sent[target], p.received[target], p.latency[target])
		if p.sent[target] == 0 || (p.sent[target] != p.received[target]) {
			t.Errorf("We are leaking packets. Sent: %d, Received: %d", p.sent[target], p.received[target])
		}
	}
}

func TestDataIntegrityValidation(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.Source = &configpb.ProbeConf_SourceIp{"1.1.1.1"}

	p, err := newProbe(c, []string{"2.2.2.2", "3.3.3.3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	tic := newTestICMPConn(c, p.targets)
	p.conn = tic

	p.runProbe()

	// We'll use sent and rcvd to take a snapshot of the probe counters.
	sent := make(map[string]int64)
	rcvd := make(map[string]int64)
	for _, target := range p.targets {
		sent[target] = p.sent[target]
		rcvd[target] = p.received[target]

		glog.Infof("target: %s, sent: %d, received: %d, total_rtt: %s", target, sent[target], rcvd[target], p.latency[target])
		if sent[target] == 0 || (sent[target] != rcvd[target]) {
			t.Errorf("We are leaking packets. Sent: %d, Received: %d", sent[target], rcvd[target])
		}
	}

	// Set the test icmp connection to flip the last byte.
	tic.setFlipLastByte()

	// Run probe again, this time we should see data integrity validation failures.
	p.runProbe()
	for _, target := range p.targets {
		glog.Infof("target: %s, sent: %d, received: %d, total_rtt: %s", target, p.sent[target], p.received[target], p.latency[target])

		// Verify that we didn't increased the received counter.
		if p.received[target] != rcvd[target] {
			t.Errorf("Unexpected change in received packets. Got: %d, Expected: %d", p.received[target], rcvd[target])
		}

		// Verify that we increased the validation failure counter.
		expectedFailures := p.sent[target] - p.received[target]
		gotFailures := p.validationFailure[target].GetKey(dataIntegrityKey).Int64()
		if p.validationFailure[target].GetKey(dataIntegrityKey).Int64() != expectedFailures {
			t.Errorf("p.validationFailure[%s].GetKey(%s)=%d, expected=%d", target, dataIntegrityKey, gotFailures, expectedFailures)
		}
	}
}
