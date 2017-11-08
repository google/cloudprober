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
)

const (
	// recv socket buffer size - we want to use a large value here, preferably
	// the maximum allowed by the OS. 425984 is the max value in
	// Container-Optimized OS version 9592.90.0.
	readBufSize = 425984
)

// ListenAndServe launches an UDP echo server listening on the configured port.
// This function returns only in case of an error.
func ListenAndServe(ctx context.Context, c *ServerConf, l *logger.Logger) error {
	conn, err := Listen(int(c.GetPort()), l)
	if err != nil {
		return err
	}
	return serve(ctx, c, conn, l)
}

// Listen opens a UDP socket on the given port. It also attempts to set recv
// buffer to a large value so that we can have many oustanding UDP messages.
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

func serve(ctx context.Context, c *ServerConf, conn *net.UDPConn, l *logger.Logger) error {
	switch c.GetType() {
	case ServerConf_ECHO:
		l.Infof("Starting UDP ECHO server on port %d", int(c.GetPort()))
		for {
			select {
			case <-ctx.Done():
				return conn.Close()
			default:
			}
			readAndEcho(conn, l)
		}
	case ServerConf_DISCARD:
		l.Infof("Starting UDP DISCARD server on port %d", int(c.GetPort()))
		for {
			select {
			case <-ctx.Done():
				return conn.Close()
			default:
			}
			readAndDiscard(conn, l)
		}
	}
	return nil
}

func readAndEcho(conn *net.UDPConn, l *logger.Logger) {
	// TODO: We read and echo back only 4098 bytes. We should look at raising this
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
