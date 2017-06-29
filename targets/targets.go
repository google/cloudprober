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

	"cloud.google.com/go/compute/metadata"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets/gce"
	"github.com/google/cloudprober/targets/lameduck"
	dnsRes "github.com/google/cloudprober/targets/resolver"
	"github.com/google/cloudprober/targets/rtc"
)

// globalResolver is a singleton DNS resolver that is used as the default
// resolver by targets. It is a singleton because dnsRes.Resolver provides a
// cache layer that is best shared by all probes.
var (
	globalResolver *dnsRes.Resolver
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
// TODO: Currently planning on refactoring lameduck. We want a lameduck
//               interface which does the work (sortof passed as a singleton). By
//               doing this, we can then test with rtcStub, and in general have
//               better "config-from-above". If I change lameduck, we can remove
//               excludeLameducks and just check if the provider is nil.
type targets struct {
	l                lister
	r                resolver
	log              *logger.Logger
	re               *regexp.Regexp
	excludeLameducks bool
}

// Resolve either resolves a target using the core resolver, or returns an error
// if no core resolver was provided. Currently all target types provide a
// resolver.
func (t *targets) Resolve(name string, ipVer int) (net.IP, error) {
	if t.r == nil {
		return nil, errors.New("no Resolver provided by this target type")
	}
	return t.r.Resolve(name, ipVer)
}

// List returns the list of targets. It gets the list of targets from the
// targets provider (instances, or forwarding rules), filters them by the
// configured regex, excludes lame ducks and returns the resultant list.
//
// This method should be concurrency safe as it doesn't modify any shared
// variables and doesn't rely on multiple accesses to same variable being
// consistent.
func (t *targets) List() []string {
	if t.l == nil {
		t.log.Error("List(): Lister t.l is nil")
		return []string{}
	}

	list := t.l.List()

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
	if t.excludeLameducks && metadata.OnGCE() {
		lameDucksList, err := lameduck.List()
		if err != nil {
			t.log.Errorf("targets.List: Error getting list of lameducking targets: %v", err)
			return list
		}
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
func baseTargets(l *logger.Logger, re string, excludeLameducks bool) (*targets, error) {
	if l == nil {
		l = &logger.Logger{}
	}
	var reg *regexp.Regexp
	var err error
	if re != "" {
		reg, err = regexp.Compile(re)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid targets regex: %s. Err: %v", re, err)
	}

	return &targets{
		re:               reg,
		excludeLameducks: excludeLameducks,
		log:              l,
		r:                globalResolver,
	}, nil
}

// StaticTargets returns a basic "targets" object (implementing the targets
// interface) from a comma-separated list of hosts.
// This function is specially useful if you want to get a valid targets object
// without an associated TargetDef protobuf (for example for testing).
func StaticTargets(hosts string) Targets {
	t, _ := baseTargets(nil, "", false)
	sl := &staticLister{}
	for _, name := range strings.Split(hosts, ",") {
		sl.list = append(sl.list, strings.TrimSpace(name))
	}
	t.l = sl
	t.r = globalResolver
	return t
}

// New returns an instance of Targets as defined by a Targets protobuf (and a
// GlobalTargetsOptions protobuf). The Targets instance returned will filter a
// core target lister (i.e. static host-list, GCE instances, GCE forwarding
// rules, RTC targets) by an optional regex or with the lameduck mechanism.
//
// All information related to creating the Target instance will be logged to
// globalTargetsLogger. The logger "l" will be given to the new Targets instance
// for future logging. If "l" is not provided, a default instance will be given.
//
// See cloudprober/targets/targets.proto for more information on the possible
// configurations of Targets.
func New(targetsDef *TargetsDef, targetOpts *GlobalTargetsOptions, globalTargetsLogger, l *logger.Logger) (Targets, error) {
	t, err := baseTargets(l, targetsDef.GetRegex(), targetsDef.GetExcludeLameducks())
	if err != nil {
		globalTargetsLogger.Error("Unable to produce the base target lister")
		return nil, fmt.Errorf("targets.New(): Error making baseTargets: %v", err)
	}

	switch targetsDef.Type.(type) {
	case *TargetsDef_HostNames:
		sl := &staticLister{}
		for _, name := range strings.Split(targetsDef.GetHostNames(), ",") {
			sl.list = append(sl.list, strings.TrimSpace(name))
		}
		t.l = sl
		t.r = globalResolver
	case *TargetsDef_GceTargets:
		proj, err := metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("targets.New(): Error getting project ID: %v", err)
		}
		s, err := gce.New(targetsDef.GetGceTargets(), targetOpts.GetGlobalGceTargetsOptions(), proj, globalResolver, globalTargetsLogger)
		if err != nil {
			l.Error("Unable to build GCE targets")
			return nil, fmt.Errorf("targets.New(): Error building GCE targets: %v", err)
		}
		t.l, t.r = s, s
	case *TargetsDef_RtcTargets:
		// TODO: we should really consolidate all these metadata calls
		// to one place.
		proj, err := metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("targets.New(): Error getting project ID: %v", err)
		}
		li, err := rtc.New(targetsDef.GetRtcTargets(), proj, l)
		if err != nil {
			return nil, fmt.Errorf("targets.New(): Error building RTC resolver: %v", err)
		}
		t.l = li
		t.r = li
	case *TargetsDef_DummyTargets:
		dummy := &dummy{}
		t.l = dummy
		t.r = dummy
	default:
		return nil, errors.New("unknown targets type")
	}

	return t, nil
}

// init initializes the package by creating a new global resolver.
func init() {
	globalResolver = dnsRes.New()
}
