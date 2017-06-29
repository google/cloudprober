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

// ListenAndServe launches an UDP echo server listening on the configured port.
// This function returns only in case of an error.
func ListenAndServe(ctx context.Context, c *ServerConf, l *logger.Logger) error {
	conn, err := listen(c, l)
	if err != nil {
		return err
	}
	return serve(ctx, c, conn, l)
}

func listen(c *ServerConf, l *logger.Logger) (*net.UDPConn, error) {
	return net.ListenUDP("udp", &net.UDPAddr{Port: int(c.GetPort())})
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
