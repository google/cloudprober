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

/*
Package iputils implements utilities to work with IP addresses.
*/
package iputils

import (
	"fmt"
	"net"
)

// IPVersion tells if an IP address is IPv4 or IPv6.
func IPVersion(ip net.IP) int {
	if len(ip.To4()) == net.IPv4len {
		return 4
	}
	if len(ip) == net.IPv6len {
		return 6
	}
	return 0
}

// Addr is used for tests, allowing net.InterfaceByName to be mocked.
type Addr interface {
	Addrs() ([]net.Addr, error)
}

// InterfaceByName is a mocking point for net.InterfaceByName, used for tests.
var InterfaceByName = func(s string) (Addr, error) { return net.InterfaceByName(s) }

// ResolveIntfAddr takes the name of a network interface and IP version, and
// returns the first IP address of the interface that matches the specified IP
// version. If no IP version is specified (ipVer is 0), simply the first IP
// address is returned.
// TODO(manugarg): This functions is currently tested through options_test. We
// should fix that.
func ResolveIntfAddr(intfName string, ipVer int) (net.IP, error) {
	i, err := InterfaceByName(intfName)
	if err != nil {
		return nil, fmt.Errorf("resolveIntfAddr(%v, %d) got error getting interface: %v", intfName, ipVer, err)
	}

	addrs, err := i.Addrs()
	if err != nil {
		return nil, fmt.Errorf("resolveIntfAddr(%v, %d) got error getting addresses for interface: %v", intfName, ipVer, err)
	} else if len(addrs) == 0 {
		return nil, fmt.Errorf("resolveIntfAddr(%v, %d) go 0 addrs for interface", intfName, ipVer)
	}

	var ip net.IP

	for _, addr := range addrs {
		// i.Addrs() mostly returns network addresses of the form "172.17.90.252/23".
		// This bit of code will pull the IP address from this address.
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			return nil, fmt.Errorf("resolveIntfAddr(%v, %d) found unknown type for first address: %T", intfName, ipVer, v)
		}

		if ipVer == 0 || IPVersion(ip) == ipVer {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("resolveIntfAddr(%v, %d) found no apprpriate IP addresses in %v", intfName, ipVer, addrs)
}
