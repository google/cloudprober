// Copyright 2018 Google Inc.
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
Package client implements a ResourceDiscovery service (RDS) client.
*/
package client

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/targets/rds/client/proto"
	pb "github.com/google/cloudprober/targets/rds/proto"
	spb "github.com/google/cloudprober/targets/rds/proto"
	"google.golang.org/grpc"
)

// Client represents an RDS based client instance.
type Client struct {
	mu    sync.Mutex
	c     *configpb.ClientConf
	cache map[string]net.IP
	names []string
	rdc   spb.ResourceDiscoveryClient
	l     *logger.Logger
}

// refreshState refreshes the client cache.
func (client *Client) refreshState() {
	response, err := client.rdc.ListResources(context.Background(), client.c.GetRequest())
	if err != nil {
		client.l.Errorf("rds.client: error getting resources from RDS server: %v", err)
		return
	}
	client.updateState(response)
}

func (client *Client) updateState(response *pb.ListResourcesResponse) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.names = make([]string, len(response.GetResources()))
	for i, res := range response.GetResources() {
		ip := net.ParseIP(res.GetIp())
		if ip == nil {
			client.l.Errorf("rds.client: errors parsing IP address for %s, IP string: %s", res.GetName(), res.GetIp())
			continue
		}
		client.cache[res.GetName()] = ip
		client.names[i] = res.GetName()
	}
}

// List returns the list of resource names.
func (client *Client) List() []string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return append([]string{}, client.names...)
}

// Resolve returns the IP address for the given resource. If no IP address is
// associated with the resource, an error is returned.
func (client *Client) Resolve(name string, ipVer int) (net.IP, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.cache[name] == nil {
		return nil, fmt.Errorf("no IP address for the resource: %s", name)
	}
	ip := client.cache[name]
	ip4 := ip.To4()
	if ipVer == 6 {
		if ip4 == nil {
			return ip, nil
		}
		return nil, fmt.Errorf("no IPv6 address (IP: %s) for %s", ip.String(), name)
	}
	if ip4 != nil {
		return ip, nil
	}
	return nil, fmt.Errorf("no IPv4 address (IP: %s) for %s", ip.String(), name)
}

// New creates an RDS (ResourceDiscovery service) client instance and set it up
// for continuous refresh.
func New(c *configpb.ClientConf, l *logger.Logger) (*Client, error) {
	conn, err := grpc.Dial(c.GetServerAddr(), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second
	client := &Client{
		c:     c,
		cache: make(map[string]net.IP),
		rdc:   spb.NewResourceDiscoveryClient(conn),
		l:     l,
	}
	go func() {
		client.refreshState()
		// Introduce a random delay between 0-reEvalInterval before starting the
		// refreshState loop. If there are multiple cloudprober instances, this will
		// make sure that each instance calls RDS server at a different point of
		// time.
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for _ = range time.Tick(reEvalInterval) {
			client.refreshState()
		}
	}()

	return client, nil
}
