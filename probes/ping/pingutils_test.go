// Copyright 2017-2019 Google Inc.
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
	"bytes"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/probes/options"
	configpb "github.com/google/cloudprober/probes/ping/proto"
	"github.com/google/cloudprober/targets"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestPreparePayload(t *testing.T) {
	ti := time.Now().UnixNano()

	timeBytes := make([]byte, 8)
	prepareRequestPayload(timeBytes, ti)

	// Verify that we send a payload that we can get the original time out of.
	for _, size := range []int{256, 1999} {
		payload := make([]byte, size)
		prepareRequestPayload(payload, ti)

		// Verify that time bytes are intact.
		ts := bytesToTime(payload)
		if ts != ti {
			t.Errorf("Got incorrect timestamp: %d, expected: %d", ts, ti)
		}
	}
}

func testPrepareRequestPacket(t *testing.T, ver int32, size int) {
	var runID, seq uint16
	runID = 12 // some number
	seq = 143  // some number
	unixNano := time.Now().UnixNano()

	// Get packet bytes using icmp.Marshal. We'll compare these bytes with the
	// bytes generated by p.prepareRequestPacket below.
	var typ icmp.Type
	typ = ipv4.ICMPTypeEcho
	if ver == 6 {
		typ = ipv6.ICMPTypeEchoRequest
	}

	payload := make([]byte, size)
	prepareRequestPayload(payload, unixNano)

	expectedPacketBytes, _ := (&icmp.Message{
		Type: typ, Code: 0,
		Body: &icmp.Echo{
			ID: int(runID), Seq: int(seq),
			Data: payload,
		},
	}).Marshal(nil)

	// Build a packet using p.prepareOutPacke
	p := &Probe{
		name: "ping_test",
		opts: &options.Options{
			ProbeConf: &configpb.ProbeConf{
				UseDatagramSocket: proto.Bool(false),
			},
			Targets:   targets.StaticTargets("test.com"),
			Interval:  2 * time.Second,
			Timeout:   time.Second,
			IPVersion: int(ver),
		},
	}
	p.initInternal()

	pktbuf := make([]byte, icmpHeaderSize+size)
	p.prepareRequestPacket(pktbuf, runID, seq, unixNano)

	if !bytes.Equal(pktbuf, expectedPacketBytes) {
		t.Errorf("p.prepareRequestPacket doesn't generate the same packet (%v) as icmp.Marshal (%v)", pktbuf, expectedPacketBytes)
	}
}

func TestPrepareRequestPacketIPv4(t *testing.T) {
	for _, size := range []int{56, 512, 1500} {
		testPrepareRequestPacket(t, 4, size)
	}
}

func TestPrepareRequestPacketIPv6(t *testing.T) {
	for _, size := range []int{56, 512, 1500} {
		testPrepareRequestPacket(t, 6, size)
	}
}

func TestPktString(t *testing.T) {
	testPkt := &rcvdPkt{
		id:     5,
		seq:    456,
		target: "test-target",
	}
	rtt := 5 * time.Millisecond
	expectedString := "peer=test-target id=5 seq=456 rtt=5ms"
	got := testPkt.String(rtt)
	if got != expectedString {
		t.Errorf("pktString(%q, %s): expected=%s wanted=%s", testPkt, rtt, got, expectedString)
	}
}
