// Copyright 2017-2020 Google Inc.
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

// +build !windows

package udp

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/google/cloudprober/logger"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func readAndEchoLoop(ctx context.Context, conn *net.UDPConn, buf []byte, l *logger.Logger) error {
	// We use an IPv6 connection wrapper to receive both IPv4 and IPv6 packets.
	// ipv6.PacketConn lets us use control messages to:
	//  -- receive packet destination IP (FlagDst)
	//  -- set source IP (Src).
	p6 := ipv6.NewPacketConn(conn)
	if err := p6.SetControlMessage(ipv6.FlagDst, true); err != nil {
		return fmt.Errorf("error running SetControlMessage(FlagDst): %v", err)
	}
	p4 := ipv4.NewPacketConn(conn)

	for {
		select {
		case <-ctx.Done():
			return conn.Close()
		default:
		}
		readAndEchoNonWindows(p6, p4, buf, l)
	}
}

func readAndEchoNonWindows(p6 *ipv6.PacketConn, p4 *ipv4.PacketConn, buf []byte, l *logger.Logger) {
	// ipv6.PacketConn also receives IPv4 packets.
	len, cm, addr, err := p6.ReadFrom(buf)
	if err != nil {
		l.Errorf("ReadFrom(): %v", err)
		return
	}

	var n int
	if cm.Dst.To4() != nil {
		// We have a v4 packet, use an ipv4.PacketConn for sending.
		wcm := &ipv4.ControlMessage{
			Src: cm.Dst.To4(),
		}
		n, err = p4.WriteTo(buf[:len], wcm, addr)
	} else {
		// We have a v6 packet.
		wcm := &ipv6.ControlMessage{
			Src: cm.Dst.To16(),
		}
		n, err = p6.WriteTo(buf[:len], wcm, addr)
	}

	if err == io.EOF {
		return
	}
	if err != nil {
		l.Errorf("WriteTo(): %v", err)
		return
	}
	if n < len {
		l.Warningf("Reply truncated! Got %v bytes but only sent %v bytes", len, n)
	}
}
