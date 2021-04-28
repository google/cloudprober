// Copyright 2020 The Cloudprober Authors.
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
//
// This file implements a client-side load balancing resolver for gRPC clients.
// This resolver takes a comma separated list of addresses and sets client
// connection to use those addresses in a round-robin manner. It implements
// the APIs defined in google.golang.org/grpc/resolver.

package client

import (
	"math/rand"
	"net"
	"strings"

	cpRes "github.com/google/cloudprober/targets/resolver"
	"google.golang.org/grpc/resolver"
)

// srvListResolver implements the resolver.Resolver interface.
type srvListResolver struct {
	hostList    []string
	portList    []string
	r           *cpRes.Resolver
	cc          resolver.ClientConn
	defaultPort string
}

func parseAddr(addr, defaultPort string) (host, port string, err error) {
	if ipStr, ok := formatIP(addr); ok {
		return ipStr, defaultPort, nil
	}

	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		return "", "", err
	}

	if port == "" {
		port = defaultPort
	}

	// target has port, i.e ipv4-host:port, [ipv6-host]:port, host-name:port
	if host == "" {
		// Keep consistent with net.Dial(): If the host is empty, as in ":80", the local system is assumed.
		host = "localhost"
	}

	return
}

// formatIP returns ok = false if addr is not a valid textual representation of an IP address.
// If addr is an IPv4 address, return the addr and ok = true.
// If addr is an IPv6 address, return the addr enclosed in square brackets and ok = true.
func formatIP(addr string) (addrIP string, ok bool) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return "", false
	}
	if ip.To4() != nil {
		return addr, true
	}
	return "[" + addr + "]", true
}

func (res *srvListResolver) resolve() (*resolver.State, error) {
	state := &resolver.State{}

	for i, host := range res.hostList {
		if ipStr, ok := formatIP(host); ok {
			state.Addresses = append(state.Addresses, resolver.Address{
				Addr: ipStr + ":" + res.portList[i],
			})
			continue
		}

		ip, err := res.r.Resolve(host, 0)
		if err != nil {
			return nil, err
		}
		state.Addresses = append(state.Addresses, resolver.Address{
			Addr: ip.String() + ":" + res.portList[i],
		})
	}

	// Set round robin policy.
	state.ServiceConfig = res.cc.ParseServiceConfig("{\"loadBalancingPolicy\": \"round_robin\"}")
	return state, nil
}

func (res *srvListResolver) ResolveNow(_ resolver.ResolveNowOptions) {
	state, err := res.resolve()
	if err != nil {
		res.cc.ReportError(err)
		return
	}

	res.cc.UpdateState(*state)
}

func (res *srvListResolver) Close() {
}

func newSrvListResolver(target, defaultPort string) (*srvListResolver, error) {
	res := &srvListResolver{
		r:           cpRes.New(),
		defaultPort: defaultPort,
	}

	addrs := strings.Split(target, ",")

	// Shuffle addresses to create variance in what order different clients start
	// connecting to these addresses. Note that round-robin load balancing policy
	// takes care of distributing load evenly over time.
	rand.Shuffle(len(addrs), func(i, j int) {
		addrs[i], addrs[j] = addrs[j], addrs[i]
	})

	for _, addr := range addrs {
		host, port, err := parseAddr(addr, defaultPort)
		if err != nil {
			return nil, err
		}

		res.hostList = append(res.hostList, host)
		res.portList = append(res.portList, port)
	}

	return res, nil
}

type srvListBuilder struct {
	defaultPort string
}

// Scheme returns the naming scheme of this resolver builder, which is "srvlist".
func (slb *srvListBuilder) Scheme() string {
	return "srvlist"
}

func (slb *srvListBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	res, err := newSrvListResolver(target.Endpoint, slb.defaultPort)
	if err != nil {
		return nil, err
	}

	res.cc = cc

	state, err := res.resolve()
	if err != nil {
		res.cc.ReportError(err)
	} else {
		res.cc.UpdateState(*state)
	}

	return res, nil
}
