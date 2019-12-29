// Copyright 2017-2019 Google Inc.
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

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/config/runconfig"
	"github.com/google/cloudprober/logger"
	rdsclient "github.com/google/cloudprober/rds/client"
	rdsclientpb "github.com/google/cloudprober/rds/client/proto"
	rdspb "github.com/google/cloudprober/rds/proto"
	"github.com/google/cloudprober/targets/endpoint"
	"github.com/google/cloudprober/targets/gce"
	"github.com/google/cloudprober/targets/lameduck"
	targetspb "github.com/google/cloudprober/targets/proto"
	dnsRes "github.com/google/cloudprober/targets/resolver"
	"github.com/google/cloudprober/targets/rtc"
)

// globalResolver is a singleton DNS resolver that is used as the default
// resolver by targets. It is a singleton because dnsRes.Resolver provides a
// cache layer that is best shared by all probes.
var (
	globalResolver *dnsRes.Resolver
)

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
	listerWithEndpoints
	resolver
}

type lister interface {
	// List produces list of targets.
	List() []string
}

// listerWithEndpoints encapsulates lister and a new method ListEndpoints().
// Note: This is a temporary interface to provide easy transition from List()
// returning []string to to List() returning []Endpoint.
type listerWithEndpoints interface {
	lister

	// ListEndpoints returns list of endpoints (name, port tupples).
	ListEndpoints() []endpoint.Endpoint
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
	list []string
}

// List returns a copy of its static host list.
func (sh *staticLister) List() []string {
	return append([]string{}, sh.list...)
}

// A dummy target object, for external probes that don't have any
// "proper" targets.
type dummy struct {
}

// List for dummy targets returns one empty string.  This is to ensure
// that any iteration over targets will at least be executed once.
func (d *dummy) List() []string {
	return []string{""}
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
	lister   lister
	resolver resolver
	re       *regexp.Regexp
	ldLister lameduck.Lister
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

func (t *targets) lameduckMap() map[string]bool {
	lameDuckMap := make(map[string]bool)
	if t.ldLister != nil {
		lameDucksList := t.ldLister.List()
		for _, i := range lameDucksList {
			lameDuckMap[i] = true
		}
	}
	return lameDuckMap
}

func (t *targets) includeInResult(name string, ldMap map[string]bool) bool {
	include := true

	// Filter by regexp
	if t.re != nil && !t.re.MatchString(name) {
		include = false
	}

	if len(ldMap) != 0 && ldMap[name] {
		include = false
	}

	return include
}

// List returns the list of targets. It gets the list of targets from the
// targets provider (instances, or forwarding rules), filters them by the
// configured regex, excludes lame ducks and returns the resultant list.
//
// This method should be concurrency safe as it doesn't modify any shared
// variables and doesn't rely on multiple accesses to same variable being
// consistent.
func (t *targets) List() []string {
	if t.lister == nil {
		t.l.Error("List(): Lister t.lister is nil")
		return []string{}
	}

	list := t.lister.List()

	ldMap := t.lameduckMap()
	if t.re != nil || len(ldMap) != 0 {
		var result []string
		for _, i := range list {
			if t.includeInResult(i, ldMap) {
				result = append(result, i)
			}
		}
		list = result
	}

	return list
}

// ListEndpoints returns the list of target endpoints, where each endpoint
// consists of a name and associated metadata like port and target labels. This
// function is similar to List() above, except for the fact that it returns a
// list of Endpoint objects instead of a list of only names.
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

	// Check if our lister supports ListEndpoint() call itself. If it doesn't,
	// create a list of Endpoint just from the names returned by the List() method
	// leaving the Port field empty.
	if epLister, ok := t.lister.(listerWithEndpoints); ok {
		list = epLister.ListEndpoints()
	} else {
		names := t.lister.List()
		list = endpoint.EndpointsFromNames(names)
	}

	ldMap := t.lameduckMap()
	if t.re != nil || len(ldMap) != 0 {
		var result []endpoint.Endpoint
		for _, i := range list {
			if t.includeInResult(i.Name, ldMap) {
				result = append(result, i)
			}
		}
		list = result
	}

	return list
}

// baseTargets constructs a targets instance with no lister or resolver. It
// provides essentially everything that the targets type wraps over its lister.
func baseTargets(targetsDef *targetspb.TargetsDef, ldLister lameduck.Lister, l *logger.Logger) (*targets, error) {
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

// StaticTargets returns a basic "targets" object (implementing the targets
// interface) from a comma-separated list of hosts.
// This function is specially useful if you want to get a valid targets object
// without an associated TargetDef protobuf (for example for testing).
func StaticTargets(hosts string) Targets {
	t, _ := baseTargets(nil, nil, nil)
	sl := &staticLister{}
	for _, name := range strings.Split(hosts, ",") {
		sl.list = append(sl.list, strings.TrimSpace(name))
	}
	t.lister = sl
	t.resolver = globalResolver
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
func New(targetsDef *targetspb.TargetsDef, ldLister lameduck.Lister, globalOpts *targetspb.GlobalTargetsOptions, globalLogger, l *logger.Logger) (Targets, error) {
	t, err := baseTargets(targetsDef, ldLister, l)
	if err != nil {
		globalLogger.Error("Unable to produce the base target lister")
		return nil, fmt.Errorf("targets.New(): Error making baseTargets: %v", err)
	}

	switch targetsDef.Type.(type) {
	case *targetspb.TargetsDef_HostNames:
		st := StaticTargets(targetsDef.GetHostNames())
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

	case *targetspb.TargetsDef_RtcTargets:
		// TODO(izzycecil): we should really consolidate all these metadata calls
		// to one place.
		proj, err := metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("targets.New(): Error getting project ID: %v", err)
		}
		rt, err := rtc.New(targetsDef.GetRtcTargets(), proj, l)
		if err != nil {
			return nil, fmt.Errorf("targets.New(): Error building RTC resolver: %v", err)
		}
		t.lister, t.resolver = rt, rt

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
	extensions := proto.RegisteredExtensions(pb)
	if len(extensions) > 1 {
		return nil, fmt.Errorf("only one extension is allowed per targets definition, got %d extensions", len(extensions))
	}
	var field int
	var desc *proto.ExtensionDesc
	// There should be only one extension in one protobuf message.
	for f, d := range extensions {
		field = int(f)
		desc = d
	}
	if desc == nil {
		return nil, errors.New("unrecognized target type")
	}
	value, err := proto.GetExtension(pb, desc)
	if err != nil {
		return nil, err
	}
	l.Infof("Extension field: %d, value: %v", field, value)
	extensionMapMu.Lock()
	defer extensionMapMu.Unlock()
	newTargetsFunc, ok := extensionMap[field]
	if !ok {
		return nil, fmt.Errorf("no targets type registered for the extension: %d", field)
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
