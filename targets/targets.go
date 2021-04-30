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

// Package targets provides the means to list and resolve targets for probers in
// the cloudprober framework. To learn more about the kinds of targets
// available, first look at the "targets.proto" to see how targets are configured.
//
// The interface targets.Targets is what actually provides the ability to list
// targets (or, more generally, resources), and is what is used by probers. The
// targets.New constructor provides a specific Target implementation that wraps
// filtering functionality over a core target lister.
package targets

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/config/runconfig"
	"github.com/google/cloudprober/logger"
	rdsclient "github.com/google/cloudprober/rds/client"
	rdsclientpb "github.com/google/cloudprober/rds/client/proto"
	rdspb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/targets/endpoint"
	"github.com/google/cloudprober/targets/file"
	"github.com/google/cloudprober/targets/gce"
	targetspb "github.com/google/cloudprober/targets/proto"
	dnsRes "github.com/google/cloudprober/targets/resolver"
)

// globalResolver is a singleton DNS resolver that is used as the default
// resolver by targets. It is a singleton because dnsRes.Resolver provides a
// cache layer that is best shared by all probes.
var (
	globalResolver *dnsRes.Resolver
)

// Targets must have refreshed this much time after the lameduck for them to
// become valid again. This is to take care of the following race between
// targets refresh and lameduck creation:
//   Targets are refreshed few seconds after lameduck, and are deleted few more
//   seconds after that. If there is no min lameduck duration, we'll end up
//   ignoring lameduck in this case. With min lameduck duration, targets will
//   need to be refreshed few minutes after being lameducked.
const minLameduckDuration = 5 * time.Minute

// extensionMap is a map of targets-types extensions. While creating new
// targets, if it's not a known targets type, we lookup this map to check if
// it matches with a registered extension.
var (
	extensionMap   = make(map[int]func(interface{}, *logger.Logger) (Targets, error))
	extensionMapMu sync.Mutex
)

var (
	sharedTargets   = make(map[string]Targets)
	sharedTargetsMu sync.RWMutex
)

// Targets are able to list and resolve targets with their List and Resolve
// methods.  A single instance of Targets represents a specific listing method
// --- if multiple sets of resources need to be listed/resolved, a separate
// instance of Targets will be needed.
type Targets interface {
	endpoint.Lister
	resolver
}

type resolver interface {
	// Resolve, given a target and IP Version will return the IP address for that
	// target.
	Resolve(name string, ipVer int) (net.IP, error)
}

// staticLister is a simple list of hosts that does not change. This corresponds
// to the "host_names" type in cloudprober/targets/targets.proto.  For
// example, one could have a probe whose targets are `host_names:
// "www.google.com, 8.8.8.8"`.
type staticLister struct {
	list []endpoint.Endpoint
}

// List returns a copy of its static host list.
func (sl *staticLister) ListEndpoints() []endpoint.Endpoint {
	return sl.list
}

// A dummy target object, for external probes that don't have any
// "proper" targets.
type dummy struct {
}

// List for dummy targets returns one empty string.  This is to ensure
// that any iteration over targets will at least be executed once.
func (d *dummy) ListEndpoints() []endpoint.Endpoint {
	return []endpoint.Endpoint{endpoint.Endpoint{Name: ""}}
}

// Resolve will just return an unspecified IP address.  This can be
// easily checked by using `func (IP) IsUnspecified`.
func (d *dummy) Resolve(name string, ipVer int) (net.IP, error) {
	return net.IPv6unspecified, nil
}

// targets is the main implementation of the Targets interface, composed of a core
// lister and resolver. Essentially it provides a wrapper around the core lister,
// providing various filtering options. Currently filtering by regex and lameduck
// is supported.
type targets struct {
	lister   endpoint.Lister
	resolver resolver
	re       *regexp.Regexp
	ldLister endpoint.Lister
	l        *logger.Logger
}

// Resolve either resolves a target using the core resolver, or returns an error
// if no core resolver was provided. Currently all target types provide a
// resolver.
func (t *targets) Resolve(name string, ipVer int) (net.IP, error) {
	if t.resolver == nil {
		return nil, errors.New("no Resolver provided by this target type")
	}
	return t.resolver.Resolve(name, ipVer)
}

func (t *targets) lameduckMap() map[string]endpoint.Endpoint {
	lameDuckMap := make(map[string]endpoint.Endpoint)
	if t.ldLister != nil {
		lameDucksList := t.ldLister.ListEndpoints()
		for _, i := range lameDucksList {
			lameDuckMap[i.Name] = i
		}
	}
	return lameDuckMap
}

func (t *targets) includeInResult(ep endpoint.Endpoint, ldMap map[string]endpoint.Endpoint) bool {
	// Filter by regexp
	if t.re != nil && !t.re.MatchString(ep.Name) {
		return false
	}

	if len(ldMap) == 0 {
		return true
	}

	// If there is a lameduck entry and it was updated after the target entry,
	// don't include it in result.
	ldEP, ok := ldMap[ep.Name]
	if ok {
		// If lameduck endpoint or target endpoint don't have last-updated set,
		// skip checking which one is newer.
		if ldEP.LastUpdated.IsZero() || ep.LastUpdated.IsZero() {
			return false
		}
		return ep.LastUpdated.After(ldEP.LastUpdated.Add(minLameduckDuration))
	}

	return true
}

// ListEndpoints returns the list of target endpoints, where each endpoint
// consists of a name and associated metadata like port and target labels.
//
// It gets the list of targets from the configured targets type, filters them
// by the configured regex, excludes lame ducks and returns the resultant list.
//
// This method should be concurrency safe as it doesn't modify any shared
// variables and doesn't rely on multiple accesses to same variable being
// consistent.
//
// Note that some targets, for example static hosts, may not have any
// associated metadata at all, those endpoint fields are left empty in that
// case.
func (t *targets) ListEndpoints() []endpoint.Endpoint {
	if t.lister == nil {
		t.l.Error("List(): Lister t.lister is nil")
		return []endpoint.Endpoint{}
	}

	var list []endpoint.Endpoint

	list = t.lister.ListEndpoints()

	ldMap := t.lameduckMap()
	if t.re != nil || len(ldMap) != 0 {
		var result []endpoint.Endpoint
		for _, ep := range list {
			if t.includeInResult(ep, ldMap) {
				result = append(result, ep)
			}
		}
		list = result
	}

	return list
}

// baseTargets constructs a targets instance with no lister or resolver. It
// provides essentially everything that the targets type wraps over its lister.
func baseTargets(targetsDef *targetspb.TargetsDef, ldLister endpoint.Lister, l *logger.Logger) (*targets, error) {
	if l == nil {
		l = &logger.Logger{}
	}

	tgts := &targets{
		l:        l,
		resolver: globalResolver,
		ldLister: ldLister,
	}

	if targetsDef == nil {
		return tgts, nil
	}

	if targetsDef.GetRegex() != "" {
		var err error
		if tgts.re, err = regexp.Compile(targetsDef.GetRegex()); err != nil {
			return nil, fmt.Errorf("invalid targets regex: %s. Err: %v", targetsDef.GetRegex(), err)
		}
	}

	return tgts, nil
}

// StaticTargets returns a basic "targets" object (implementing the Targets
// interface) from a comma-separated list of hosts. This function panics if
// "hosts" string is not valid. It is mainly used by tests to quickly get a
// targets.Targets object from a list of hosts.
func StaticTargets(hosts string) Targets {
	t, err := staticTargets(hosts)
	if err != nil {
		panic(err)
	}
	return t
}

// RDSClientConf converts RDS targets into RDS client configuration.
func RDSClientConf(pb *targetspb.RDSTargets, globalOpts *targetspb.GlobalTargetsOptions, l *logger.Logger) (rdsclient.ListResourcesFunc, *rdsclientpb.ClientConf, error) {
	var listResourcesFunc rdsclient.ListResourcesFunc

	// Intialize server address with global options.
	serverOpts := globalOpts.GetRdsServerOptions()
	if serverOpts == nil && globalOpts.GetRdsServerAddress() != "" {
		l.Warning("rds_server_address is now deprecated. please use rds_server_options instead")
		serverOpts = &rdsclientpb.ClientConf_ServerOptions{
			ServerAddress: proto.String(globalOpts.GetRdsServerAddress()),
		}
	}

	// If targets specific rds_server_options is given, use that.
	if pb.GetRdsServerOptions() != nil {
		serverOpts = pb.GetRdsServerOptions()
	}

	// If rds_server_address is not given in both, local options and in global
	// options, look for the locally running RDS server.
	if serverOpts == nil {
		localRDSServer := runconfig.LocalRDSServer()
		if localRDSServer == nil {
			return nil, nil, fmt.Errorf("rds_server_address not given and found no local RDS server")
		}
		listResourcesFunc = localRDSServer.ListResources
	}

	toks := strings.SplitN(pb.GetResourcePath(), "://", 2)
	if len(toks) != 2 || toks[0] == "" {
		return nil, nil, fmt.Errorf("provider not specified in the resource_path: %s", pb.GetResourcePath())
	}
	provider := toks[0]

	return listResourcesFunc, &rdsclientpb.ClientConf{
		ServerOptions: serverOpts,
		Request: &rdspb.ListResourcesRequest{
			Provider:     &provider,
			ResourcePath: &toks[1],
			Filter:       pb.GetFilter(),
			IpConfig:     pb.GetIpConfig(),
		},
	}, nil
}

// New returns an instance of Targets as defined by a Targets protobuf (and a
// GlobalTargetsOptions protobuf). The Targets instance returned will filter a
// core target lister (i.e. static host-list, GCE instances, GCE forwarding
// rules, RTC targets) by an optional regex or with the lameduck mechanism.
//
// All information related to creating the Target instance will be logged to
// globalLogger. The logger "l" will be given to the new Targets instance
// for future logging. If "l" is not provided, a default instance will be given.
//
// See cloudprober/targets/targets.proto for more information on the possible
// configurations of Targets.
func New(targetsDef *targetspb.TargetsDef, ldLister endpoint.Lister, globalOpts *targetspb.GlobalTargetsOptions, globalLogger, l *logger.Logger) (Targets, error) {
	t, err := baseTargets(targetsDef, ldLister, l)
	if err != nil {
		globalLogger.Error("Unable to produce the base target lister")
		return nil, fmt.Errorf("targets.New(): Error making baseTargets: %v", err)
	}

	switch targetsDef.Type.(type) {
	case *targetspb.TargetsDef_HostNames:
		st, err := staticTargets(targetsDef.GetHostNames())
		if err != nil {
			return nil, fmt.Errorf("targets.New(): error creating targets from host_names: %v", err)
		}
		t.lister, t.resolver = st, st

	case *targetspb.TargetsDef_SharedTargets:
		sharedTargetsMu.RLock()
		defer sharedTargetsMu.RUnlock()
		st := sharedTargets[targetsDef.GetSharedTargets()]
		if st == nil {
			return nil, fmt.Errorf("targets.New(): Shared targets %s are not defined", targetsDef.GetSharedTargets())
		}
		t.lister, t.resolver = st, st

	case *targetspb.TargetsDef_GceTargets:
		s, err := gce.New(targetsDef.GetGceTargets(), globalOpts.GetGlobalGceTargetsOptions(), globalResolver, globalLogger)
		if err != nil {
			return nil, fmt.Errorf("targets.New(): error creating GCE targets: %v", err)
		}
		t.lister, t.resolver = s, s

	case *targetspb.TargetsDef_RdsTargets:
		listResourcesFunc, clientConf, err := RDSClientConf(targetsDef.GetRdsTargets(), globalOpts, l)
		if err != nil {
			return nil, fmt.Errorf("target.New(): error creating RDS client: %v", err)
		}

		client, err := rdsclient.New(clientConf, listResourcesFunc, l)
		if err != nil {
			return nil, fmt.Errorf("target.New(): error creating RDS client: %v", err)
		}

		t.lister, t.resolver = client, client

	case *targetspb.TargetsDef_FileTargets:
		ft, err := file.New(targetsDef.GetFileTargets(), globalResolver, l)
		if err != nil {
			return nil, fmt.Errorf("target.New(): %v", err)
		}
		t.lister, t.resolver = ft, ft

	case *targetspb.TargetsDef_DummyTargets:
		dummy := &dummy{}
		t.lister, t.resolver = dummy, dummy

	default:
		extT, err := getExtensionTargets(targetsDef, t.l)
		if err != nil {
			return nil, fmt.Errorf("targets.New(): %v", err)
		}
		t.lister, t.resolver = extT, extT
	}

	return t, nil
}

func getExtensionTargets(pb *targetspb.TargetsDef, l *logger.Logger) (Targets, error) {
	extensions, err := proto.ExtensionDescs(pb)
	if err != nil {
		return nil, fmt.Errorf("error getting extensions from the target config (%s): %v", pb.String(), err)
	}
	if len(extensions) != 1 {
		return nil, fmt.Errorf("there should be exactly one extension in the targets config (%s), got %d extensions", pb.String(), len(extensions))
	}
	desc := extensions[0]
	value, err := proto.GetExtension(pb, desc)
	if err != nil {
		return nil, err
	}
	l.Infof("Extension field: %d, value: %v", desc.Field, value)
	extensionMapMu.Lock()
	defer extensionMapMu.Unlock()
	newTargetsFunc, ok := extensionMap[int(desc.Field)]
	if !ok {
		return nil, fmt.Errorf("no targets type registered for the extension: %d", desc.Field)
	}
	return newTargetsFunc(value, l)
}

// RegisterTargetsType registers a new targets type. New targets types are
// integrated with the config subsystem using the protobuf extensions.
//
// TODO(manugarg): Add a full example of using extensions.
func RegisterTargetsType(extensionFieldNo int, newTargetsFunc func(interface{}, *logger.Logger) (Targets, error)) {
	extensionMapMu.Lock()
	defer extensionMapMu.Unlock()
	extensionMap[extensionFieldNo] = newTargetsFunc
}

// SetSharedTargets adds given targets to an internal map. These targets can
// then be referred by multiple probes through "shared_targets" option.
func SetSharedTargets(name string, tgts Targets) {
	sharedTargetsMu.Lock()
	defer sharedTargetsMu.Unlock()
	sharedTargets[name] = tgts
}

// init initializes the package by creating a new global resolver.
func init() {
	globalResolver = dnsRes.New()
}
