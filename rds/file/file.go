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
	"google.golang.org/protobuf/proto"
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

	lastUpdated  time.Time
	checkModTime bool
}

func (ls *lister) lastModified() int64 {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.lastUpdated.Unix()
}

// listResources returns the last successfully parsed list of resources.
func (ls *lister) listResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	// If there are no filters, return early.
	if len(req.GetFilter()) == 0 {
		return &pb.ListResourcesResponse{
			Resources:    append([]*pb.Resource{}, ls.resources...),
			LastModified: proto.Int64(ls.lastUpdated.Unix()),
		}, nil
	}

	allFilters, err := filter.ParseFilters(req.GetFilter(), SupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}
	nameFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.LabelsFilter

	// Allocate resources for response early but optimize for large number of
	// total resources.
	allocSize := len(ls.resources)
	if allocSize > 100 {
		allocSize = 100
	}
	resources := make([]*pb.Resource, 0, allocSize)

	for _, res := range ls.resources {
		if nameFilter != nil && !nameFilter.Match(res.GetName(), ls.l) {
			continue
		}
		if labelsFilter != nil && !labelsFilter.Match(res.GetLabels(), ls.l) {
			continue
		}
		resources = append(resources, res)
	}

	ls.l.Infof("file.ListResources: returning %d resources out of %d", len(resources), len(ls.resources))
	return &pb.ListResourcesResponse{
		Resources:    resources,
		LastModified: proto.Int64(ls.lastUpdated.Unix()),
	}, nil
}

func (ls *lister) parseFileContent(b []byte) ([]*pb.Resource, error) {
	resources := &configpb.FileResources{}

	switch ls.format {
	case configpb.ProviderConfig_TEXTPB:
		err := prototext.Unmarshal(b, resources)
		if err != nil {
			return nil, fmt.Errorf("file_provider(%s): error unmarshaling as text proto: %v", ls.filePath, err)
		}
		return resources.GetResource(), nil
	case configpb.ProviderConfig_JSON:
		err := protojson.Unmarshal(b, resources)
		if err != nil {
			return nil, fmt.Errorf("file_provider(%s): error unmarshaling as JSON: %v", ls.filePath, err)
		}
		return resources.GetResource(), nil
	}

	return nil, fmt.Errorf("file_provider(%s): unknown format - %v", ls.filePath, ls.format)
}

func (ls *lister) shouldReloadFile() bool {
	if !ls.checkModTime {
		return true
	}

	modTime, err := file.ModTime(ls.filePath)
	if err != nil {
		ls.l.Warningf("file(%s): Error getting modified time: %v; Ignoring modified time check.", ls.filePath, err)
		return true
	}

	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return modTime.After(ls.lastUpdated)
}

func (ls *lister) refresh() error {
	if !ls.shouldReloadFile() {
		ls.l.Infof("file(%s): Skipping reloading file as it has not changed since its last refresh at %v", ls.filePath, ls.lastUpdated)
		return nil
	}

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

	ls.lastUpdated = time.Now()
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
func newLister(filePath string, c *configpb.ProviderConfig, l *logger.Logger) (*lister, error) {
	format := c.GetFormat()
	if format == configpb.ProviderConfig_UNSPECIFIED {
		format = formatFromPath(filePath)
		l.Infof("file_provider: Determined file format from file name: %v", format)
	}

	ls := &lister{
		filePath:     filePath,
		format:       format,
		l:            l,
		checkModTime: !c.GetDisableModifiedTimeCheck(),
	}

	reEvalSec := c.GetReEvalSec()
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

func responseWithCacheCheck(ls *lister, req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	if req.GetIfModifiedSince() == 0 {
		return ls.listResources(req)
	}

	if lastModified := ls.lastModified(); lastModified <= req.GetIfModifiedSince() {
		return &pb.ListResourcesResponse{
			LastModified: proto.Int64(lastModified),
		}, nil
	}

	return ls.listResources(req)
}

// ListResources returns the list of resources based on the given request.
func (p *Provider) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	fPath := req.GetResourcePath()
	if fPath != "" {
		ls := p.listers[fPath]
		if ls == nil {
			return nil, fmt.Errorf("file path %s is not available on this server", fPath)
		}
		return responseWithCacheCheck(ls, req)
	}

	// Avoid append and another allocation if there is only one lister, most
	// common use case.
	if len(p.listers) == 1 {
		for _, ls := range p.listers {
			return responseWithCacheCheck(ls, req)
		}
	}

	// If we are working with multiple listers, it's slightly more complicated.
	// In that case we need to return all the listers' resources even if only one
	// of them has changed.
	//
	// Get the latest last-modified.
	lastModified := int64(0)
	for _, ls := range p.listers {
		listerLastModified := ls.lastModified()
		if lastModified < listerLastModified {
			lastModified = listerLastModified
		}
	}
	resp := &pb.ListResourcesResponse{
		LastModified: proto.Int64(lastModified),
	}

	// if nothing changed since req.IfModifiedSince, return early.
	if req.GetIfModifiedSince() != 0 && lastModified <= req.GetIfModifiedSince() {
		return resp, nil
	}

	var result []*pb.Resource
	for _, fp := range p.filePaths {
		res, err := p.listers[fp].listResources(req)
		if err != nil {
			return nil, err
		}
		result = append(result, res.Resources...)
	}
	resp.Resources = result
	return resp, nil
}

// Provider provides a file-based targets provider for RDS. It implements the
// RDS server's Provider interface.
type Provider struct {
	filePaths []string
	listers   map[string]*lister
}

// New creates a File (file) provider for RDS server, based on the
// provided config.
func New(c *configpb.ProviderConfig, l *logger.Logger) (*Provider, error) {
	filePaths := c.GetFilePath()
	p := &Provider{
		filePaths: filePaths,
		listers:   make(map[string]*lister),
	}

	for _, filePath := range filePaths {
		lister, err := newLister(filePath, c, l)
		if err != nil {
			return nil, err
		}
		p.listers[filePath] = lister
	}

	return p, nil
}
