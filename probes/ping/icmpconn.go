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

package ping

import (
	"net"
	"time"

	"golang.org/x/net/icmp"
)

type icmpConn interface {
	read(buf []byte) (n int, peer net.Addr, err error)
	write(buf []byte, peer net.Addr) (int, error)
	setReadDeadline(deadline time.Time)
	close()
}

type icmpPacketConn struct {
	c *icmp.PacketConn
}

func newICMPConn(proto, source string) (icmpConn, error) {
	c, err := icmp.ListenPacket(proto, source)
	if err != nil {
		return nil, err
	}
	return &icmpPacketConn{c}, nil
}

func (ipc *icmpPacketConn) read(buf []byte) (int, net.Addr, error) {
	return ipc.c.ReadFrom(buf)
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
