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
	"io"
	"net"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/servers/udp/proto"
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
}

// New returns an UDP server.
func New(initCtx context.Context, c *configpb.ServerConf, l *logger.Logger) (*Server, error) {
	conn, err := Listen(int(c.GetPort()), l)
	if err != nil {
		return nil, err
	}
	go func() {
		<-initCtx.Done()
		conn.Close()
	}()
	return &Server{
		c:    c,
		conn: conn,
		l:    l,
	}, nil
}

// Listen opens a UDP socket on the given port. It also attempts to set recv
// buffer to a large value so that we can have many outstanding UDP messages.
// Listen is exported only because it's used by udp probe tests.
func Listen(port int, l *logger.Logger) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
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
			readAndEcho(s.conn, s.l)
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

func readAndEcho(conn *net.UDPConn, l *logger.Logger) {
	// TODO(manugarg): We read and echo back only 4098 bytes. We should look at raising this
	// limit or making it configurable. Also of note, ReadFromUDP reads a single UDP datagram
	// (up to the max size of 64K-sizeof(UDPHdr)) and discards the rest.
	buf := make([]byte, 4098)
	len, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		l.Errorf("ReadFromUDP: %v", err)
		return
	}

	n, err := conn.WriteToUDP(buf[:len], addr)
	if err == io.EOF {
		return
	}
	if err != nil {
		l.Errorf("WriteToUDP: %v", err)
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
