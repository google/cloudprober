// Copyright 2021 The Cloudprober Authors.
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
Package file implements a file-based targets provider for cloudprober.
*/
package file

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/cloudprober/common/file"
	"github.com/google/cloudprober/logger"
	configpb "github.com/google/cloudprober/rds/file/proto"
	pb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/filter"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
)

// DefaultProviderID is the povider id to use for this provider if a provider
// id is not configured explicitly.
const DefaultProviderID = "file"

/*
SupportedFilters defines filters supported by the file-based resources
type.
 Example:
 filter {
	 key: "name"
	 value: "cloudprober.*"
 }
 filter {
	 key: "labels.app"
	 value: "service-a"
 }
*/
var SupportedFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	[]string{"name"},
	true,
}

// lister implements file-based targets lister.
type lister struct {
	mu        sync.RWMutex
	filePath  string
	format    configpb.ProviderConfig_Format
	resources []*pb.Resource
	l         *logger.Logger
}

// ListResources returns the last successfully parsed list of resources.
func (ls *lister) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	allFilters, err := filter.ParseFilters(req.GetFilter(), SupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}
	nameFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.LabelsFilter

	ls.mu.RLock()
	defer ls.mu.RUnlock()

	resources := make([]*pb.Resource, len(ls.resources))
	i := 0
	for _, res := range ls.resources {
		if nameFilter != nil && !nameFilter.Match(res.GetName(), ls.l) {
			continue
		}
		if labelsFilter != nil && !labelsFilter.Match(res.GetLabels(), ls.l) {
			continue
		}
		resources[i] = res
		i++
	}
	resources = resources[:i]

	ls.l.Infof("file.ListResources: returning %d resources out of %d", len(resources), len(ls.resources))
	return &pb.ListResourcesResponse{Resources: resources}, nil
}

func (ls *lister) parseFileContent(b []byte) ([]*pb.Resource, error) {
	resources := &configpb.FileResources{}

	switch ls.format {
	case configpb.ProviderConfig_TEXTPB:
		err := prototext.Unmarshal(b, resources)
		if err != nil {
			return nil, fmt.Errorf("file_provider(%s): error unmarshaling as text proto: %v", ls.filePath, err)
		}
	case configpb.ProviderConfig_JSON:
		err := protojson.Unmarshal(b, resources)
		if err != nil {
			return nil, fmt.Errorf("file_provider(%s): error unmarshaling as JSON: %v", ls.filePath, err)
		}
	default:
		return nil, fmt.Errorf("file_provider(%s): unknown format - %v", ls.filePath, ls.format)
	}

	return resources.GetResource(), nil
}

func (ls *lister) refresh() error {
	b, err := file.ReadFile(ls.filePath)
	if err != nil {
		return fmt.Errorf("file(%s): error while reading file: %v", ls.filePath, err)
	}

	resources, err := ls.parseFileContent(b)
	if err != nil {
		return err
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.resources = resources

	ls.l.Infof("file_provider(%s): Read %d resources.", ls.filePath, len(ls.resources))
	return nil
}

func formatFromPath(path string) configpb.ProviderConfig_Format {
	switch filepath.Ext(path) {
	case ".textpb":
		return configpb.ProviderConfig_TEXTPB
	case ".json":
		return configpb.ProviderConfig_JSON
	}
	return configpb.ProviderConfig_TEXTPB
}

// newLister creates a new file-based targets lister.
func newLister(filePath string, format configpb.ProviderConfig_Format, reEvalSec int32, l *logger.Logger) (*lister, error) {
	if format == configpb.ProviderConfig_UNSPECIFIED {
		format = formatFromPath(filePath)
		l.Infof("file_provider: Determined file format from file name: %v", format)
	}

	ls := &lister{
		filePath: filePath,
		format:   format,
		l:        l,
	}

	if reEvalSec == 0 {
		return ls, ls.refresh()
	}

	reEvalInterval := time.Duration(reEvalSec) * time.Second
	go func() {
		if err := ls.refresh(); err != nil {
			l.Error(err.Error())
		}
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// instances, this will make sure that each instance refreshes
		// at a different point of time.
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			if err := ls.refresh(); err != nil {
				l.Error(err.Error())
			}
		}
	}()

	return ls, nil
}

// ListResources returns the list of resources based on the given request.
func (p *Provider) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	fPath := req.GetResourcePath()
	if fPath != "" {
		ls := p.listers[fPath]
		if ls == nil {
			return nil, fmt.Errorf("file path %s is not available on this server", fPath)
		}
		return ls.ListResources(req)
	}

	var result []*pb.Resource
	for _, fp := range p.filePaths {
		res, err := p.listers[fp].ListResources(req)
		if err != nil {
			return nil, err
		}
		result = append(result, res.Resources...)
	}

	return &pb.ListResourcesResponse{Resources: result}, nil
}

// Provider provides a file-based targets provider for RDS.
type Provider struct {
	filePaths []string
	listers   map[string]*lister
}

// New creates a Kubernetes (k8s) provider for RDS server, based on the
// provided config.
func New(c *configpb.ProviderConfig, l *logger.Logger) (*Provider, error) {
	filePaths := c.GetFilePath()
	p := &Provider{
		filePaths: filePaths,
		listers:   make(map[string]*lister),
	}

	for _, filePath := range filePaths {
		lister, err := newLister(filePath, c.GetFormat(), c.GetReEvalSec(), l)
		if err != nil {
			return nil, err
		}
		p.listers[filePath] = lister
	}

	return p, nil
}
