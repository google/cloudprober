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

// +build windows

package udp

import (
	"io"
	"net"

	"github.com/google/cloudprober/logger"
)

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
