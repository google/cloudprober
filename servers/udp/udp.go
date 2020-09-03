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

/*
Package udp implements a UDP server.  It listens on a
given port and echos whatever it receives.  This is used for the UDP probe.
*/
package udp

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/servers/udp/proto"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	// recv socket buffer size - we want to use a large value here, preferably
	// the maximum allowed by the OS. 425984 is the max value in
	// Container-Optimized OS version 9592.90.0.
	readBufSize = 425984
)

// Server implements a basic UDP server.
type Server struct {
	c    *configpb.ServerConf
	conn *net.UDPConn
	p6   *ipv6.PacketConn
	p4   *ipv4.PacketConn
	l    *logger.Logger
}

// New returns an UDP server.
func New(initCtx context.Context, c *configpb.ServerConf, l *logger.Logger) (*Server, error) {
	conn, err := Listen(&net.UDPAddr{Port: int(c.GetPort())}, l)
	if err != nil {
		return nil, err
	}
	go func() {
		<-initCtx.Done()
		conn.Close()
	}()

	// Set up PacketConn wrappers around the connection, to receive packet
	// destination IPs (FlagDst) and to specify source IPs with control messages.
	// See readAndEcho() for usage.
	p6 := ipv6.NewPacketConn(conn)
	p4 := ipv4.NewPacketConn(conn)
	if err := p6.SetControlMessage(ipv6.FlagDst, true); err != nil {
		return nil, fmt.Errorf("SetControlMessage(FlagDst): %v", err)
	}

	return &Server{
		c:    c,
		conn: conn,
		p4:   p4,
		p6:   p6,
		l:    l,
	}, nil
}

// Listen opens a UDP socket on the given port. It also attempts to set recv
// buffer to a large value so that we can have many outstanding UDP messages.
// Listen is exported only because it's used by udp probe tests.
func Listen(addr *net.UDPAddr, l *logger.Logger) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	if err = conn.SetReadBuffer(readBufSize); err != nil {
		// Non-fatal error if we are not able to set read socket buffer.
		l.Errorf("Error setting UDP socket %v read buffer to %d: %s. Continuing...",
			conn.LocalAddr(), readBufSize, err)
	}

	return conn, nil
}

// Start starts the UDP server. It returns only when context is canceled.
func (s *Server) Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error {
	switch s.c.GetType() {
	case configpb.ServerConf_ECHO:
		s.l.Infof("Starting UDP ECHO server on port %d", int(s.c.GetPort()))
		for {
			select {
			case <-ctx.Done():
				return s.conn.Close()
			default:
			}
			readAndEcho(s.p6, s.p4, s.l)
		}
	case configpb.ServerConf_DISCARD:
		s.l.Infof("Starting UDP DISCARD server on port %d", int(s.c.GetPort()))
		for {
			select {
			case <-ctx.Done():
				return s.conn.Close()
			default:
			}
			readAndDiscard(s.conn, s.l)
		}
	}
	return nil
}

func readAndEcho(p6 *ipv6.PacketConn, p4 *ipv4.PacketConn, l *logger.Logger) {
	// TODO(manugarg): We read and echo back only 4098 bytes. We should look at raising this
	// limit or making it configurable. Also of note, ReadFromUDP reads a single UDP datagram
	// (up to the max size of 64K-sizeof(UDPHdr)) and discards the rest.
	buf := make([]byte, 4098)

	// ipv6.PacketConn also receives IPv4 packets.
	len, cm, addr, err := p6.ReadFrom(buf)
	if err != nil {
		l.Errorf("ReadFrom(): %v", err)
		return
	}

	var n int
	if cm.Dst.To4() != nil {
		// We have a v4 packet, but need to use an ipv4.PacketConn for sending.
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

// listenAndServeDiscard launches an UDP discard server listening on port.
func readAndDiscard(conn *net.UDPConn, l *logger.Logger) {
	buf := make([]byte, 4098)
	_, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		l.Errorf("ReadFromUDP: %v", err)
	}
}
