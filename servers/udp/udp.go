// Copyright 2017-2020 The Cloudprober Authors.
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
	"runtime"
	"strings"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/servers/udp/proto"
	"golang.org/x/net/ipv6"
)

const (
	// recv socket buffer size - we want to use a large value here, preferably
	// the maximum allowed by the OS. 425984 is the max value in
	// Container-Optimized OS version 9592.90.0.
	readBufSize = 425984

	// Number of messages to batch read.
	batchSize = 16

	// Maximum packet size.
	// TODO(manugarg): We read and echo back only 4098 bytes. We should look at raising this
	// limit or making it configurable. Also of note, ReadFromUDP reads a single UDP datagram
	// (up to the max size of 64K-sizeof(UDPHdr)) and discards the rest.
	maxPacketSize = 4098
)

// Server implements a basic UDP server.
type Server struct {
	c    *configpb.ServerConf
	conn *net.UDPConn
	l    *logger.Logger

	p6 *ipv6.PacketConn
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

		// We use an IPv6 connection wrapper to receive both IPv4 and IPv6 packets.
		// ipv6.PacketConn lets us use control messages (non-Windows only) to:
		//  -- receive packet destination IP (FlagDst)
		//  -- set source IP (Src).
		p6: ipv6.NewPacketConn(conn),
	}

	switch runtime.GOOS {
	case "windows":
		// Control messages are not supported.
	default:
		if err := s.p6.SetControlMessage(ipv6.FlagDst, true); err != nil {
			return nil, fmt.Errorf("SetControlMessage(ipv6.FlagDst, true) failed: %v", err)
		}
	}

	return s, nil
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

func (s *Server) writePacketsBatch(ms []ipv6.Message) error {
	for remaining := len(ms); remaining > 0; {
		n, err := s.p6.WriteBatch(ms, 0)
		if err != nil {
			return fmt.Errorf("writing packets: %v", err)
		}
		if n == 0 {
			return fmt.Errorf("wrote zero packets, %d remain", remaining)
		}
		remaining -= n
	}
	return nil
}

func (s *Server) readAndEchoWindows(buf []byte) error {
	inLen, addr, err := s.conn.ReadFromUDP(buf)
	if err != nil {
		return fmt.Errorf("error reading from UDP: %v", err)
	}

	n, err := s.conn.WriteToUDP(buf[:inLen], addr)
	if err != nil {
		return fmt.Errorf("error writing to UDP: %v", err)
	}

	if n < inLen {
		s.l.Warningf("Reply truncated! Got %d bytes but only sent %d bytes", inLen, n)
	}
	return nil
}

// readAndEcho reads a packet from the server connection and writes it back.
// To determine the source address for outgoing packets (e.g. if server
// is behind a load balancer), we make use of control messages (Note: this
// doesn't work on Windows OS as Go doesn't provide a way to access control
// messages on Windows).
//
// Note that we don't need to copy or modify the messages below before echoing
// them for the following reasons:
//   - Message struct uses the same field (Addr) for sender (while receiving)
//     and destination address (while sending).
//   - Control message (type: packet-info) field that contains the received
//     packet's destination address, is also the field that's used to set the
//     source address on the outgoing packets.
func (s *Server) readAndEcho(ms []ipv6.Message, buf []byte) error {
	if runtime.GOOS == "windows" {
		return s.readAndEchoWindows(buf)
	}

	n, err := s.p6.ReadBatch(ms, 0)
	if err != nil {
		return fmt.Errorf("error reading from UDP: %v", err)
	}
	ms = ms[:n]

	// Resize buffers to match amount read.
	for _, m := range ms {
		// We only allocated a 0th buffer, so all the data is there.
		m.Buffers[0] = m.Buffers[0][:m.N]
	}

	err = s.writePacketsBatch(ms)
	if err != nil {
		return fmt.Errorf("error writing to UDP: %v", err)
	}

	// Reset buffers to full size for re-use.
	for _, m := range ms {
		b := m.Buffers[0]
		// We only allocated a 0th buffer.
		m.Buffers[0] = b[:cap(b)]
	}

	return nil
}

func (s *Server) readAndDiscard(ms []ipv6.Message, buf []byte) (err error) {
	if runtime.GOOS == "windows" {
		_, _, err = s.conn.ReadFromUDP(buf)
		return
	}
	_, err = s.p6.ReadBatch(ms, 0)
	return
}

func connClosed(err error) bool {
	// TODO(manugarg): Replace this by errors.Is(err, net.ErrClosed) once Go 1.16
	// is more widely available.
	return strings.Contains(err.Error(), "use of closed network connection")
}

// Start starts the UDP server. It returns only when context is canceled.
func (s *Server) Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error {
	ms := make([]ipv6.Message, batchSize)
	for i := 0; i < batchSize; i++ {
		ms[i].Buffers = [][]byte{make([]byte, maxPacketSize)}
		switch runtime.GOOS {
		case "linux":
			ms[i].OOB = ipv6.NewControlMessage(ipv6.FlagDst)
		default:
			// Control messages are not supported.
		}
	}

	// buf is used for higher level read-wrie funtions on Windows.
	var buf []byte

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
			if err := s.readAndEcho(ms, buf); err != nil {
				if connClosed(err) {
					s.l.Warning("connection closed, stopping the start goroutine")
					return nil
				}
				s.l.Error(err.Error())
			}
		}

	case configpb.ServerConf_DISCARD:
		s.l.Infof("Starting UDP DISCARD server on port %d", int(s.c.GetPort()))
		for {
			if err := s.readAndDiscard(ms, buf); err != nil {
				if connClosed(err) {
					return nil
				}
				s.l.Errorf("ReadFromUDP: %v", err)
			}
		}
	}

	return nil
}
