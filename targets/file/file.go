// Copyright 2020-2021 The Cloudprober Authors.
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
Package file implements a file-based targets for cloudprober.
*/
package file

import (
	"context"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/rds/client"
	client_configpb "github.com/google/cloudprober/rds/client/proto"
	"github.com/google/cloudprober/rds/file"
	file_configpb "github.com/google/cloudprober/rds/file/proto"
	rdspb "github.com/google/cloudprober/rds/proto"
	configpb "github.com/google/cloudprober/targets/file/proto"
	dnsRes "github.com/google/cloudprober/targets/resolver"
)

// New returns new file targets.
func New(opts *configpb.TargetsConf, res *dnsRes.Resolver, l *logger.Logger) (*client.Client, error) {
	lister, err := file.New(&file_configpb.ProviderConfig{
		FilePath: []string{opts.GetFilePath()},
	}, l)
	if err != nil {
		return nil, err
	}

	clientConf := &client_configpb.ClientConf{
		Request:   &rdspb.ListResourcesRequest{Filter: opts.GetFilter()},
		ReEvalSec: opts.ReEvalSec,
	}

	return client.New(clientConf, func(_ context.Context, req *rdspb.ListResourcesRequest) (*rdspb.ListResourcesResponse, error) {
		return lister.ListResources(req)
	}, l)
}
