// Copyright 2017 Google Inc.
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
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets/gce"
	"github.com/google/cloudprober/targets/lameduck"
	targetspb "github.com/google/cloudprober/targets/proto"
	rdsclient "github.com/google/cloudprober/targets/rds/client"
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
	lister
	resolver
}

type lister interface {
	// List produces list of targets.
	List() []string
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

// List returns a copy of its static host list. As currently implemented this
// returns the same slice it was initialized with. It is the responsibility of
// the caller not to modify its value.
func (sh *staticLister) List() []string {
	// Todo(tcecil): Shouldn't this be returning a COPY of sh.list, for correctness
	//               sake?
	// c := make([]string, len(sh.list))
	// copy(c, sh.list)
	return sh.list
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

	// Filter by regexp
	if t.re != nil {
		var filter []string
		for _, i := range list {
			if t.re.MatchString(i) {
				filter = append(filter, i)
			}
		}
		list = filter
	}

	// Filter by lameduck
	if t.ldLister != nil {
		lameDucksList := t.ldLister.List()

		lameDuckMap := make(map[string]bool)
		for _, i := range lameDucksList {
			lameDuckMap[i] = true
		}
		var filter []string
		for _, i := range list {
			if !lameDuckMap[i] {
				filter = append(filter, i)
			}
		}
		list = filter
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
func New(targetsDef *targetspb.TargetsDef, ldLister lameduck.Lister, targetOpts *targetspb.GlobalTargetsOptions, globalLogger, l *logger.Logger) (Targets, error) {
	t, err := baseTargets(targetsDef, ldLister, l)
	if err != nil {
		globalLogger.Error("Unable to produce the base target lister")
		return nil, fmt.Errorf("targets.New(): Error making baseTargets: %v", err)
	}

	switch targetsDef.Type.(type) {
	case *targetspb.TargetsDef_HostNames:
		sl := &staticLister{}
		for _, name := range strings.Split(targetsDef.GetHostNames(), ",") {
			sl.list = append(sl.list, strings.TrimSpace(name))
		}
		t.lister = sl
		t.resolver = globalResolver

	case *targetspb.TargetsDef_SharedTargets:
		sharedTargetsMu.RLock()
		defer sharedTargetsMu.RUnlock()
		st := sharedTargets[targetsDef.GetSharedTargets()]
		if st == nil {
			return nil, fmt.Errorf("targets.New(): Shared targets %s are not defined", targetsDef.GetSharedTargets())
		}
		t.lister, t.resolver = st, st

	case *targetspb.TargetsDef_GceTargets:
		s, err := gce.New(targetsDef.GetGceTargets(), targetOpts.GetGlobalGceTargetsOptions(), globalResolver, globalLogger)
		if err != nil {
			l.Error("Unable to build GCE targets")
			return nil, fmt.Errorf("targets.New(): Error building GCE targets: %v", err)
		}
		t.lister, t.resolver = s, s

	case *targetspb.TargetsDef_RdsTargets:
		li, err := rdsclient.New(targetsDef.GetRdsTargets(), l)
		if err != nil {
			return nil, fmt.Errorf("targets.New(): Error building RDS targets: %v", err)
		}
		t.lister = li
		t.resolver = li

	case *targetspb.TargetsDef_RtcTargets:
		// TODO(izzycecil): we should really consolidate all these metadata calls
		// to one place.
		proj, err := metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("targets.New(): Error getting project ID: %v", err)
		}
		li, err := rtc.New(targetsDef.GetRtcTargets(), proj, l)
		if err != nil {
			return nil, fmt.Errorf("targets.New(): Error building RTC resolver: %v", err)
		}
		t.lister = li
		t.resolver = li

	case *targetspb.TargetsDef_DummyTargets:
		dummy := &dummy{}
		t.lister = dummy
		t.resolver = dummy

	default:
		extT, err := getExtensionTargets(targetsDef, t.l)
		if err != nil {
			return nil, fmt.Errorf("targets.New(): %v", err)
		}
		t.lister = extT
		t.resolver = extT
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
