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
	"fmt"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (s *Server) initConnection() error {
	// We use an IPv6 connection wrapper to receive both IPv4 and IPv6 packets.
	// ipv6.PacketConn lets us use control messages to:
	//  -- receive packet destination IP (FlagDst)
	//  -- set source IP (Src).
	s.p6 = ipv6.NewPacketConn(s.conn)
	if err := s.p6.SetControlMessage(ipv6.FlagDst, true); err != nil {
		return fmt.Errorf("error running SetControlMessage(FlagDst): %v", err)
	}
	s.p4 = ipv4.NewPacketConn(s.conn)

	return nil
}
