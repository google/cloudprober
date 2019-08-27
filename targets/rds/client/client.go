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
	"google.golang.org/grpc/credentials"
)

// Client represents an RDS based client instance.
type Client struct {
	mu            sync.Mutex
	c             *configpb.ClientConf
	cache         map[string]net.IP
	names         []string
	listResources func(context.Context, *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error)
	l             *logger.Logger
}

// ListResourcesFunc is a function that takes ListResourcesRequest and returns
// ListResourcesResponse.
type ListResourcesFunc func(context.Context, *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error)

// refreshState refreshes the client cache.
func (client *Client) refreshState(timeout time.Duration) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()

	response, err := client.listResources(ctx, client.c.GetRequest())
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
		var ip net.IP

		if res.GetIp() != "" {
			ip = net.ParseIP(res.GetIp())
			if ip == nil {
				client.l.Errorf("rds.client: errors parsing IP address for %s, IP string: %s", res.GetName(), res.GetIp())
				continue
			}
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

	// If we don't care about IP version, return whatever we've got.
	if ipVer == 0 {
		return ip, nil
	}

	// Verify that the IP matches the version we need.
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

func (client *Client) grpcListResources() error {
	dialOpts := grpc.WithInsecure()
	if client.c.GetTlsCertFile() != "" {
		creds, err := credentials.NewClientTLSFromFile(client.c.GetTlsCertFile(), "")
		if err != nil {
			return err
		}
		dialOpts = grpc.WithTransportCredentials(creds)
	}

	conn, err := grpc.Dial(client.c.GetServerAddr(), dialOpts)
	if err != nil {
		return err
	}

	client.listResources = func(ctx context.Context, in *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
		return spb.NewResourceDiscoveryClient(conn).ListResources(ctx, in)
	}
	return nil
}

// New creates an RDS (ResourceDiscovery service) client instance and set it up
// for continuous refresh.
func New(c *configpb.ClientConf, listResources ListResourcesFunc, l *logger.Logger) (*Client, error) {
	client := &Client{
		c:             c,
		cache:         make(map[string]net.IP),
		listResources: listResources,
		l:             l,
	}

	// If listResources is not provided, use gRPC client's.
	if client.listResources == nil {
		err := client.grpcListResources()
		if err != nil {
			return nil, err
		}
	}

	reEvalInterval := time.Duration(client.c.GetReEvalSec()) * time.Second
	client.refreshState(reEvalInterval)
	go func() {
		// Introduce a random delay between 0-reEvalInterval before starting the
		// refreshState loop. If there are multiple cloudprober instances, this will
		// make sure that each instance calls RDS server at a different point of
		// time.
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			client.refreshState(reEvalInterval)
		}
	}()

	return client, nil
}
