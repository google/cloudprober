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

package ping

import (
	"fmt"
	"net"
	"time"

	"github.com/google/cloudprober/probes/probeutils"
)

const timeBytesSize = 8

func timeToBytes(t time.Time, size int) []byte {
	nsec := t.UnixNano()
	var timeBytes [timeBytesSize]byte
	for i := uint8(0); i < timeBytesSize; i++ {
		// To get timeBytes:
		// 0th byte - shift bits by 56 (7*8) bits, AND with 0xff to get the last 8 bits
		// 1st byte - shift bits by 48 (6*8) bits, AND with 0xff to get the last 8 bits
		// ... ...
		// 7th byte - shift bits by 0 (0*8) bits, AND with 0xff to get the last 8 bits
		timeBytes[i] = byte((nsec >> ((timeBytesSize - i - 1) * timeBytesSize)) & 0xff)
	}
	return probeutils.PatternPayload(timeBytes[:], size)
}

// verifyPayload verifies that in the provided byte array first 8-bytes are
// repeated for the rest array, except for the last "len(array) mod 8 bytes".
func verifyPayload(b []byte) error {
	// Since we set the pattern ourselves in timeToBytes, we know that the pattern
	// is 8-bytes long (timestamp).
	return probeutils.VerifyPayloadPattern(b, b[:8])
}

func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < timeBytesSize; i++ {
		nsec += int64(b[i]) << ((timeBytesSize - i - 1) * timeBytesSize)
	}
	return time.Unix(0, nsec)
}

// ipVersion tells if an IP address is IPv4 or IPv6.
func ipVersion(ip net.IP) int {
	if len(ip.To4()) == net.IPv4len {
		return 4
	}
	if len(ip) == net.IPv6len {
		return 6
	}
	return 0
}

// resolveAddr resolves a host name into an IP address.
func resolveAddr(t string, ver int) (net.IP, error) {
	if ip := net.ParseIP(t); ip != nil {
		if ipVersion(ip) == ver {
			return ip, nil
		}
		return nil, fmt.Errorf("IP (%s) is not an IPv%d address", ip, ver)
	}
	ips, err := net.LookupIP(t)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if ver == ipVersion(ip) {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no good IPs found for the ip version (%d). IPs found: %q", ver, ips)
}
