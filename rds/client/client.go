// Copyright 2018-2020 Google Inc.
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
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/cloudprober/common/oauth"
	"github.com/google/cloudprober/common/tlsconfig"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/client/proto"
	pb "github.com/google/cloudprober/rds/proto"
	spb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/targets/endpoint"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcoauth "google.golang.org/grpc/credentials/oauth"
)

type cacheRecord struct {
	ip     net.IP
	port   int
	labels map[string]string
}

// Default RDS port
const defaultRDSPort = "9314"

// Client represents an RDS based client instance.
type Client struct {
	mu            sync.Mutex
	c             *configpb.ClientConf
	serverOpts    *configpb.ClientConf_ServerOptions
	dialOpts      []grpc.DialOption
	cache         map[string]*cacheRecord
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
		client.cache[res.GetName()] = &cacheRecord{ip, int(res.GetPort()), res.Labels}
		client.names[i] = res.GetName()
	}
}

// List returns the list of resource names.
func (client *Client) List() []string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return append([]string{}, client.names...)
}

// ListEndpoints returns the list of resources.
func (client *Client) ListEndpoints() []endpoint.Endpoint {
	client.mu.Lock()
	defer client.mu.Unlock()
	result := make([]endpoint.Endpoint, len(client.names))
	for i, name := range client.names {
		result[i] = endpoint.Endpoint{Name: name, Port: client.cache[name].port, Labels: client.cache[name].labels}
	}
	return result
}

// Resolve returns the IP address for the given resource. If no IP address is
// associated with the resource, an error is returned.
func (client *Client) Resolve(name string, ipVer int) (net.IP, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	cr, ok := client.cache[name]
	if !ok || cr.ip == nil {
		return nil, fmt.Errorf("no IP address for the resource: %s", name)
	}
	ip := cr.ip

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

func (client *Client) connect(serverAddr string) (*grpc.ClientConn, error) {
	client.l.Infof("rds.client: using RDS servers at: %s", serverAddr)

	if strings.HasPrefix(serverAddr, "srvlist:///") {
		client.dialOpts = append(client.dialOpts, grpc.WithResolvers(&srvListBuilder{defaultPort: defaultRDSPort}))
	}

	return grpc.Dial(client.serverOpts.GetServerAddress(), client.dialOpts...)
}

// initListResourcesFunc uses server options to establish a connection with the
// given RDS server.
func (client *Client) initListResourcesFunc() error {
	if client.listResources != nil {
		return nil
	}

	if client.serverOpts == nil || client.serverOpts.GetServerAddress() == "" {
		return errors.New("rds.Client: RDS server address not defined")
	}

	// Transport security options.
	if client.serverOpts.GetTlsConfig() != nil {
		tlsConfig := &tls.Config{}
		if err := tlsconfig.UpdateTLSConfig(tlsConfig, client.serverOpts.GetTlsConfig(), false); err != nil {
			return err
		}
		client.dialOpts = append(client.dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		client.dialOpts = append(client.dialOpts, grpc.WithInsecure())
	}

	// OAuth related options.
	if client.serverOpts.GetOauthConfig() != nil {
		oauthTS, err := oauth.TokenSourceFromConfig(client.serverOpts.GetOauthConfig(), client.l)
		if err != nil {
			return err
		}
		client.dialOpts = append(client.dialOpts, grpc.WithPerRPCCredentials(grpcoauth.TokenSource{oauthTS}))
	}

	conn, err := client.connect(client.serverOpts.GetServerAddress())
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
		serverOpts:    c.GetServerOptions(),
		cache:         make(map[string]*cacheRecord),
		listResources: listResources,
		l:             l,
	}

	if err := client.initListResourcesFunc(); err != nil {
		return nil, err
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
