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

/*
Package file implements a file-based targets for cloudprober.
*/
package file

import (
	"fmt"
	"math/rand"
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/cloudprober/common/file"
	"github.com/google/cloudprober/common/iputils"
	"github.com/google/cloudprober/logger"
	rdspb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/rds/server/filter"
	"github.com/google/cloudprober/targets/endpoint"
	configpb "github.com/google/cloudprober/targets/file/proto"
	dnsRes "github.com/google/cloudprober/targets/resolver"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
)

// Targets encapsulates file based targets.
type Targets struct {
	path   string
	format configpb.TargetsConf_Format
	r      *dnsRes.Resolver

	names        []string
	resources    map[string]*rdspb.Resource
	nameFilter   *filter.RegexFilter
	labelsFilter *filter.LabelsFilter
	mu           sync.RWMutex
	l            *logger.Logger
}

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

// New returns new file targets.
func New(opts *configpb.TargetsConf, res *dnsRes.Resolver, l *logger.Logger) (*Targets, error) {
	ft := &Targets{
		path:      opts.GetFilePath(),
		resources: make(map[string]*rdspb.Resource),
		r:         res,
		l:         l,
	}

	ft.format = opts.GetFormat()
	if ft.format == configpb.TargetsConf_UNSPECIFIED {
		ft.format = ft.formatFromPath()
		ft.l.Infof("file_targets: Determined file format from file name: %v", ft.format)
	}

	allFilters, err := filter.ParseFilters(opts.GetFilter(), SupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	ft.nameFilter, ft.labelsFilter = allFilters.RegexFilters["name"], allFilters.LabelsFilter

	if opts.GetReEvalSec() == 0 {
		return ft, ft.refresh()
	}

	reEvalInterval := time.Duration(opts.GetReEvalSec()) * time.Second
	go func() {
		if err := ft.refresh(); err != nil {
			ft.l.Error(err.Error())
		}
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// instances, this will make sure that each instance refreshes
		// at a different point of time.
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			if err := ft.refresh(); err != nil {
				ft.l.Error(err.Error())
			}
		}
	}()

	return ft, nil
}

func (ft *Targets) formatFromPath() configpb.TargetsConf_Format {
	switch filepath.Ext(ft.path) {
	case ".textpb":
		return configpb.TargetsConf_TEXTPB
	case ".json":
		return configpb.TargetsConf_JSON
	}
	return configpb.TargetsConf_TEXTPB
}

// ListEndpoints returns the list of endpoints in the file based targets.
func (ft *Targets) ListEndpoints() []endpoint.Endpoint {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	endpoints := make([]endpoint.Endpoint, len(ft.names))
	for i, name := range ft.names {
		endpoints[i] = endpoint.Endpoint{
			Name:   name,
			Labels: ft.resources[name].GetLabels(),
			Port:   int(ft.resources[name].GetPort()),
		}
	}

	return endpoints
}

func (ft *Targets) refresh() error {
	b, err := file.ReadFile(ft.path)
	if err != nil {
		return fmt.Errorf("file_targets(%s): error while reading file: %v", ft.path, err)
	}
	return ft.parseFileContent(b)
}

func (ft *Targets) parseFileContent(b []byte) error {
	resources := &configpb.FileResources{}

	switch ft.format {
	case configpb.TargetsConf_TEXTPB:
		err := prototext.Unmarshal(b, resources)
		if err != nil {
			return fmt.Errorf("file_targets(%s): error unmarshaling as text proto: %v", ft.path, err)
		}
	case configpb.TargetsConf_JSON:
		err := protojson.Unmarshal(b, resources)
		if err != nil {
			return fmt.Errorf("file_targets(%s): error unmarshaling as JSON: %v", ft.path, err)
		}
	default:
		return fmt.Errorf("file_targets(%s): unknown format - %v", ft.path, ft.format)
	}

	// Update state.
	ft.mu.Lock()
	defer ft.mu.Unlock()

	ft.names = make([]string, len(resources.GetResource()))

	i := 0
	for _, resource := range resources.GetResource() {
		name := resource.GetName()
		if ft.nameFilter != nil && !ft.nameFilter.Match(name, ft.l) {
			continue
		}
		if ft.labelsFilter != nil && !ft.labelsFilter.Match(resource.GetLabels(), ft.l) {
			continue
		}
		ft.resources[name] = resource
		ft.names[i] = name
		i++
	}
	ft.names = ft.names[:i]

	ft.l.Infof("file_targets(%s): Read %d resources, kept %d after filtering", ft.path, len(resources.GetResource()), len(ft.names))
	return nil
}

// Resolve returns the IP address for the given resource. If no IP address is
// configured in the file, DNS resolver is used to find the IP address.
func (ft *Targets) Resolve(name string, ipVer int) (net.IP, error) {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	res := ft.resources[name]
	if res == nil {
		return nil, fmt.Errorf("file_targets(%s): Resource %s not found", ft.path, name)
	}

	// Use global resolver if file doesn't have IP address.
	if res.GetIp() == "" {
		if ft.r == nil {
			return nil, fmt.Errorf("file_targets(%s): IP not configured for %s", ft.path, name)
		}
		return ft.r.Resolve(name, ipVer)
	}

	ip := net.ParseIP(res.GetIp())
	if ip == nil {
		return nil, fmt.Errorf("file_targets(%s): Error while parsing IP %s for %s", ft.path, res.GetIp(), name)
	}

	if ipVer == 0 || iputils.IPVersion(ip) == ipVer {
		return ip, nil
	}

	return nil, fmt.Errorf("file_targets(%s): No IPv%d address (IP: %s) for %s", ft.path, ipVer, ip.String(), name)
}
