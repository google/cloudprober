// Copyright 2020 Google Inc.
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

// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package ping

import (
	"bytes"
	"encoding/binary"
	"net"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

// NativeEndian is the machine native endian implementation of ByteOrder.
var NativeEndian binary.ByteOrder

const (
	protocolIP = 0
)

func sockaddr(sourceIP net.IP, ipVer int) (syscall.Sockaddr, error) {
	a := &net.IPAddr{IP: sourceIP}

	// If sourceIP is unspecified, we bind to the 0 IP address (all).
	if sourceIP == nil {
		a.IP = map[int]net.IP{4: net.IPv4zero, 6: net.IPv6unspecified}[ipVer]
	}

	switch ipVer {
	case 4:
		sa := &syscall.SockaddrInet4{}
		copy(sa.Addr[:], a.IP)
		return sa, nil
	case 6:
		sa := &syscall.SockaddrInet6{}
		copy(sa.Addr[:], a.IP)
		return sa, nil
	default:
		return nil, net.InvalidAddrError("unexpected family")
	}
}

// listenPacket listens for incoming ICMP packets addressed to sourceIP.
// We need to write our own listenPacket instead of using "net.ListenPacket"
// for the following reasons:
//   1. ListenPacket doesn't support ICMP for SOCK_DGRAM sockets. You create
//      datagram sockets by specifying network as "udp", but UDP new connection
//      implementation ignores the protocol field entirely.
//   2. ListenPacket doesn't support setting socket options (we need
//      SO_TIMESTAMP) in a straightforward way.
func listenPacket(sourceIP net.IP, ipVer int, datagramSocket bool) (*icmpPacketConn, error) {
	var family, proto int

	switch ipVer {
	case 4:
		family, proto = syscall.AF_INET, protocolICMP
	case 6:
		family, proto = syscall.AF_INET6, protocolIPv6ICMP
	}

	sockType := syscall.SOCK_RAW
	if datagramSocket {
		sockType = syscall.SOCK_DGRAM
	}

	s, err := syscall.Socket(family, sockType, proto)
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}

	if runtime.GOOS == "darwin" && family == syscall.AF_INET {
		// 0x17 = IP_STRIPHDR -- this is required for darwin IPv4.
		if err := syscall.SetsockoptInt(s, protocolIP, 0x17, 1); err != nil {
			syscall.Close(s)
			return nil, os.NewSyscallError("setsockopt", err)
		}
	}

	// Set socket option to receive kernel's timestamp from each packet.
	// Ref: https://man7.org/linux/man-pages/man7/socket.7.html (SO_TIMESTAMP)
	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_TIMESTAMP, 1); err != nil {
		syscall.Close(s)
		return nil, os.NewSyscallError("setsockopt", err)
	}

	sa, err := sockaddr(sourceIP, ipVer)
	if err != nil {
		syscall.Close(s)
		return nil, err
	}
	if err := syscall.Bind(s, sa); err != nil {
		syscall.Close(s)
		return nil, os.NewSyscallError("bind", err)
	}

	// FilePacketConn is you get a PacketConn from a socket descriptor.
	// Behind the scene, FilePacketConn creates either an IPConn or UDPConn,
	// based on socket's local address (it gets that from the fd).
	f := os.NewFile(uintptr(s), "icmp")
	c, cerr := net.FilePacketConn(f)
	f.Close()

	if cerr != nil {
		syscall.Close(s)
		return nil, cerr
	}

	ipc := &icmpPacketConn{c: c}
	ipc.ipConn, _ = c.(*net.IPConn)
	ipc.udpConn, _ = c.(*net.UDPConn)

	return ipc, nil
}

type icmpPacketConn struct {
	c net.PacketConn

	// We use ipConn and udpConn for reading OOB data from the connection.
	ipConn  *net.IPConn
	udpConn *net.UDPConn
}

func (ipc *icmpPacketConn) read(buf []byte) (n int, addr net.Addr, recvTime time.Time, err error) {
	// We need to convert to IPConn/UDPConn so that we can read out-of-band data
	// using ReadMsg<IP,UDP> functions. PacketConn interface doesn't have method
	// that exposes OOB data.
	oob := make([]byte, 64)
	var oobn int
	if ipc.ipConn != nil {
		n, oobn, _, addr, err = ipc.ipConn.ReadMsgIP(buf, oob)
	}
	if ipc.udpConn != nil {
		n, oobn, _, addr, err = ipc.udpConn.ReadMsgUDP(buf, oob)
	}

	if err != nil {
		return
	}

	cmsgs, err := syscall.ParseSocketControlMessage(oob[:oobn])

	for _, m := range cmsgs {
		// We are interested only in socket-level control messages
		// (syscall.SOL_SOCKET)
		if m.Header.Level != syscall.SOL_SOCKET {
			continue
		}

		// SCM_TIMESTAMP is the type of the timestamp control message.
		// Note that syscall.SO_TIMESTAMP == syscall.SCM_TIMESTAMP for linux, but
		// that doesn't have to be true for other operating systems, e.g. Mac OS X.
		if m.Header.Type == syscall.SCM_TIMESTAMP {
			var tv syscall.Timeval
			binary.Read(bytes.NewReader(m.Data), NativeEndian, &tv)
			recvTime = time.Unix(tv.Unix()) // syscall.Timeval -> time.Time
		}
	}

	return
}

// write writes the ICMP message b to dst.
func (ipc *icmpPacketConn) write(buf []byte, dst net.Addr) (int, error) {
	return ipc.c.WriteTo(buf, dst)
}

// Close closes the endpoint.
func (ipc *icmpPacketConn) close() {
	ipc.c.Close()
}

// setReadDeadline sets the read deadline associated with the
// endpoint.
func (ipc *icmpPacketConn) setReadDeadline(t time.Time) {
	ipc.c.SetReadDeadline(t)
}

func newICMPConn(sourceIP net.IP, ipVer int, datagramSocket bool) (*icmpPacketConn, error) {
	return listenPacket(sourceIP, ipVer, datagramSocket)
}

// Find out native endianness when this packages is loaded.
// This code is based on:
// https://github.com/golang/net/blob/master/internal/socket/sys.go
func init() {
	i := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&i))
	if b[0] == 1 {
		NativeEndian = binary.LittleEndian
	} else {
		NativeEndian = binary.BigEndian
	}
}
