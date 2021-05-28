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

package ping

import (
	"encoding/binary"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/cloudprober/probes/probeutils"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const timeBytesSize = 8

func bytesToTime(b []byte) int64 {
	var unixNano int64
	for i := uint8(0); i < timeBytesSize; i++ {
		unixNano += int64(b[i]) << ((timeBytesSize - i - 1) * timeBytesSize)
	}
	return unixNano
}

func ipToKey(ip net.IP) (key [16]byte) {
	copy(key[:], ip.To16())
	return
}

func validEchoReply(ipVer int, typeByte byte) bool {
	switch ipVer {
	case 6:
		return ipv6.ICMPType(typeByte) == ipv6.ICMPTypeEchoReply
	default:
		return ipv4.ICMPType(typeByte) == ipv4.ICMPTypeEchoReply
	}
}

func prepareRequestPayload(payload []byte, unixNano int64) {
	var timeBytes [timeBytesSize]byte
	for i := uint8(0); i < timeBytesSize; i++ {
		// To get timeBytes:
		// 0th byte - shift bits by 56 (7*8) bits, AND with 0xff to get the last 8 bits
		// 1st byte - shift bits by 48 (6*8) bits, AND with 0xff to get the last 8 bits
		// ... ...
		// 7th byte - shift bits by 0 (0*8) bits, AND with 0xff to get the last 8 bits
		timeBytes[i] = byte((unixNano >> ((timeBytesSize - i - 1) * timeBytesSize)) & 0xff)
	}
	probeutils.PatternPayload(payload, timeBytes[:])
}

// This function is a direct copy of checksum from the following package:
// https://godoc.org/golang.org/x/net/icmp
// TODO(manugarg): Follow up to find out if checksum from icmp package can be
// made public.
func checksum(b []byte) uint16 {
	csumcv := len(b) - 1 // checksum coverage
	s := uint32(0)
	for i := 0; i < csumcv; i += 2 {
		s += uint32(b[i+1])<<8 | uint32(b[i])
	}
	if csumcv&1 == 0 {
		s += uint32(b[csumcv])
	}
	s = s>>16 + s&0xffff
	s = s + s>>16
	return ^uint16(s)
}

// prepareRequestPacket prepares the Echo request packet in the given pktbuf
// bytes buffer.
func (p *Probe) prepareRequestPacket(pktbuf []byte, runID, seq uint16, unixNano int64) {
	if p.ipVer == 6 {
		pktbuf[0] = byte(ipv6.ICMPTypeEchoRequest)
	} else {
		pktbuf[0] = byte(ipv4.ICMPTypeEcho)
	}
	pktbuf[1] = byte(0)

	// We fill these 2 bytes later with the checksum.
	pktbuf[2] = 0
	pktbuf[3] = 0

	binary.BigEndian.PutUint16(pktbuf[4:6], uint16(runID))
	binary.BigEndian.PutUint16(pktbuf[6:8], uint16(seq))

	// Fill payload with the bytes corresponding to current time.
	prepareRequestPayload(pktbuf[8:], unixNano)

	// For IPv6 checksum is always computed by the kernel.
	// For IPv4, we compute checksum only if using RAW socket or OS is darwin.
	if p.ipVer == 4 {
		if !p.useDatagramSocket || runtime.GOOS == "darwin" {
			csum := checksum(pktbuf)
			pktbuf[2] ^= byte(csum)
			pktbuf[3] ^= byte(csum >> 8)
		}
	}
}

func (pkt *rcvdPkt) String(rtt time.Duration) string {
	var b strings.Builder
	b.WriteString("peer=")
	b.WriteString(pkt.target)
	b.WriteString(" id=")
	b.WriteString(strconv.FormatInt(int64(pkt.id), 10))
	b.WriteString(" seq=")
	b.WriteString(strconv.FormatInt(int64(pkt.seq), 10))
	b.WriteString(" rtt=")
	b.WriteString(rtt.String())
	return b.String()
}
