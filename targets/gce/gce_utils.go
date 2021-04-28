// Copyright 2017-2019 The Cloudprober Authors.
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

package gce

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	rdspb "github.com/google/cloudprober/rds/proto"
	configpb "github.com/google/cloudprober/targets/gce/proto"
	dnsRes "github.com/google/cloudprober/targets/resolver"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// defaultComputeService returns the compute service to use for the GCE API
// calls.
func defaultComputeService(apiVersion string) (*compute.Service, error) {
	client, err := google.DefaultClient(context.TODO(), compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	cs, err := compute.New(client)
	if err != nil {
		return nil, err
	}
	cs.BasePath = "https://www.googleapis.com/compute/" + apiVersion + "/projects/"
	return cs, nil
}

// Utility function to parse "<key>:<value>" style strings into RDS style
// label filters.
func parseLabels(sl []string) ([]*rdspb.Filter, error) {
	var result []*rdspb.Filter
	for _, s := range sl {
		tokens := strings.Split(s, ":")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid label definition: %s", s)
		}
		result = append(result, &rdspb.Filter{
			Key:   proto.String("labels." + tokens[0]),
			Value: proto.String(tokens[1]),
		})
	}
	return result, nil
}

// Instances resource type related utilities.
// ========================================

func instancesIPConfig(ipb *configpb.Instances) *rdspb.IPConfig {
	ipTypeMap := map[string]rdspb.IPConfig_IPType{
		"PRIVATE": rdspb.IPConfig_DEFAULT,
		"PUBLIC":  rdspb.IPConfig_PUBLIC,
		"ALIAS":   rdspb.IPConfig_ALIAS,
	}

	return &rdspb.IPConfig{
		NicIndex: proto.Int32(ipb.GetNetworkInterface().GetIndex()),
		IpType:   ipTypeMap[ipb.GetNetworkInterface().GetIpType().String()].Enum(),
	}
}

func verifyInstancesConfig(ipb *configpb.Instances, globalResolver *dnsRes.Resolver) error {
	if ipb.GetUseDnsToResolve() {
		if ipb.GetNetworkInterface() != nil {
			return errors.New("network_intf and use_dns_to_resolve are mutually exclusive")
		}
		if globalResolver == nil {
			return errors.New("use_dns_to_resolve configured, but globalResolver is nil")
		}
	}
	return nil
}
