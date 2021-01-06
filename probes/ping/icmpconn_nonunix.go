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

// +build !aix,!darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris

package ping

import (
	"net"
	"strconv"
	"time"

	"golang.org/x/net/icmp"
)

type icmpPacketConn struct {
	c *icmp.PacketConn
}

func newICMPConn(sourceIP net.IP, ipVer int, datagramSocket bool) (icmpConn, error) {
	network := map[int]string{
		4: "ip4:icmp",
		6: "ip6:ipv6-icmp",
	}[ipVer]

	if datagramSocket {
		network = "udp" + strconv.Itoa(ipVer)
	}

	c, err := icmp.ListenPacket(network, sourceIP.String())
	if err != nil {
		return nil, err
	}
	return &icmpPacketConn{c}, nil
}

func (ipc *icmpPacketConn) read(buf []byte) (int, net.Addr, time.Time, error) {
	n, addr, err := ipc.c.ReadFrom(buf)
	return n, addr, time.Now(), err
}

func (ipc *icmpPacketConn) write(buf []byte, peer net.Addr) (int, error) {
	return ipc.c.WriteTo(buf, peer)
}

func (ipc *icmpPacketConn) setReadDeadline(deadline time.Time) {
	ipc.c.SetReadDeadline(deadline)
}

func (ipc *icmpPacketConn) close() {
	ipc.c.Close()
}
