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

/*
Package udp implements a UDP server.  It listens on a
given port and echos whatever it receives.  This is used for the UDP probe.
*/
package udp

import (
	"context"
	"fmt"
	"net"
	"strings"

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
	l    *logger.Logger

	// IPv4 and IPv6 connection types are initialized and used only on
	// non-windows systems.
	p6 *ipv6.PacketConn
	p4 *ipv4.PacketConn
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

	s := &Server{
		c:    c,
		conn: conn,
		l:    l,
	}

	return s, s.initConnection()
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

func (s *Server) readPacket(buf []byte) (int, *ipv6.ControlMessage, net.Addr, error) {
	// If protocol level packet connection is available (work only on
	// non-windows), use low-level read.
	if s.p6 == nil {
		len, addr, err := s.conn.ReadFromUDP(buf)
		return len, nil, addr, err
	}
	return s.p6.ReadFrom(buf)
}

func (s *Server) writePacket(buf []byte, cm *ipv6.ControlMessage, addr net.Addr) (int, error) {
	// If no control message (should happen only on windows), use high-level UDP
	// write function.
	if cm == nil {
		return s.conn.WriteToUDP(buf, addr.(*net.UDPAddr))
	}

	// We have an IPv4 address.
	if cm.Dst.To4() != nil {
		wcm := &ipv4.ControlMessage{
			Src: cm.Dst.To4(),
		}
		return s.p4.WriteTo(buf, wcm, addr)
	}

	// Assume IPv6 address.
	wcm := &ipv6.ControlMessage{
		Src: cm.Dst.To16(),
	}
	return s.p6.WriteTo(buf, wcm, addr)
}

// readAndEcho reads a packet from the server connection and writes it back. To
// determine the source address for outgoing packets (e.g. if server is behind
// a load balancer), we make use of control messages (Note: this doesn't work
// on Windows OS as Go doesn't provide a way to access control messages on
// Windows).
func (s *Server) readAndEcho(buf []byte) (error, error) {
	inLen, cm, addr, err := s.readPacket(buf)
	if err != nil {
		return fmt.Errorf("error reading from UDP: %v", err), err
	}

	n, err := s.writePacket(buf[:inLen], cm, addr)
	if err != nil {
		return fmt.Errorf("error writing to UDP: %v", err), err
	}

	if n < inLen {
		s.l.Warningf("Reply truncated! Got %d bytes but only sent %d bytes", inLen, n)
	}

	return nil, nil
}

func connClosed(err error) bool {
	// ReadFromUDP returns net.OpError
	if opErr, ok := err.(*net.OpError); ok {
		err = opErr.Err
	}
	// TODO(manugarg): Replace this by errors.Is(err, net.ErrClosed) once Go 1.16
	// is more widely available.
	return strings.Contains(err.Error(), "use of closed network connection")
}

// Start starts the UDP server. It returns only when context is canceled.
func (s *Server) Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error {
	// TODO(manugarg): We read and echo back only 4098 bytes. We should look at raising this
	// limit or making it configurable. Also of note, ReadFromUDP reads a single UDP datagram
	// (up to the max size of 64K-sizeof(UDPHdr)) and discards the rest.
	buf := make([]byte, 4098)

	// Setup a background function to close connection if context is canceled.
	// Typically, this is not what we want (close something started outside of
	// Start function), but in case of UDP we don't have better control than
	// this. One thing we can consider is to re-setup connection in Start().
	go func() {
		<-ctx.Done()
		s.conn.Close()
	}()

	switch s.c.GetType() {

	case configpb.ServerConf_ECHO:
		s.l.Infof("Starting UDP ECHO server on port %d", int(s.c.GetPort()))
		for {
			if err, nestedErr := s.readAndEcho(buf); err != nil {
				if connClosed(nestedErr) {
					s.l.Warning("connection closed, stopping the start goroutine")
					return nil
				}
				s.l.Error(err.Error())
			}
		}

	case configpb.ServerConf_DISCARD:
		s.l.Infof("Starting UDP DISCARD server on port %d", int(s.c.GetPort()))
		for {
			if _, _, err := s.conn.ReadFromUDP(buf); err != nil {
				if connClosed(err) {
					return nil
				}
				s.l.Errorf("ReadFromUDP: %v", err)
			}
		}
	}

	return nil
}
