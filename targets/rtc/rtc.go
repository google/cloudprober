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

// Package rtc implements runtime-configurator (RTC) based targets.
package rtc

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets/rtc/rtcreporter"
	"github.com/google/cloudprober/targets/rtc/rtcservice"
	runtimeconfig "google.golang.org/api/runtimeconfig/v1beta1"
)

// Targets implements the lister interface to provide RTC based target listing.
// Provides a means of listing targets stored in a Runtime Configuration value
// set. For information on how the RTC API works, see
// https://cloud.google.com/deployment-manager/runtime-configurator/reference/rest/
//
// Each variable in RTC is a key/val pair. When listing targets, we assume the
// variables in the rtcservice.Config are laid out as <Hostname : RtcTargetInfo>
// where RtcTargetInfo is a base64 encoded RtcTargetInfo protobuf. The hostnames
// are listed, with one of their addresses assigned as a solve address. If
// groupTag is empty, no filtering by groupTag occurs. Otherwise, only variables
// with a group_tag matching an element of groupTag will be listed.
//
// Note that this only provides a means to list targets stored in an RTC
// config. It does not provide means to maintain this list. When listing,
// however, Targets will ignore sufficiently old RTC entries, as configured by
// "exp_msec" in the TargetsConf protobuf.
type Targets struct {
	rtc rtcservice.Config
	l   *logger.Logger
	// Expire is a filter on the UpdateTime field of runtimeconfig.Variable.
	expire time.Duration
	// groups is a filter on the group_tag of a variables protobuf.
	groups map[string]bool
	// addrTag is the address which an rtc target should resolve to.
	addrTag string

	cacheMu     sync.RWMutex
	cache       []string
	cacheTicker *time.Ticker

	// nameToIP maps keys (hostnames) to a single address, to use Targets as
	// a resolver.
	nameToIPMu sync.RWMutex
	nameToIP   map[string]net.IP
}

// timeNow allows for dependency injection when testing Targets.List.
var timeNow = time.Now

// Resolve returns the address that was associated with a given hostname. ipVer
// is currently ignored.
func (t *Targets) Resolve(name string, ipVer int) (net.IP, error) {
	t.nameToIPMu.RLock()
	ip, ok := t.nameToIP[name]
	t.nameToIPMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("Resolve(%v, %v): Unknown name %q", name, ipVer, name)
	}
	return ip, nil
}

// List produces a list of targets found in the RTC configuration for this
// project. It ignores all sufficiently old RTC entries, as configured by
// "exp_msec" in the TargetsConf protobuf. Then entries will be filtered by group
// names, if "groups" was non-empty. If there is an empty intersection
// between "groups" and the groups in RtcTargetInfo, the entry will be filtered out.
func (t *Targets) List() []string {
	if t.cacheTicker == nil {
		return t.evalList()
	}
	t.cacheMu.RLock()
	defer t.cacheMu.RUnlock()
	return append([]string{}, t.cache...)
}

// runc preiodically re-fills the targets cache.
func (t *Targets) runc() {
	t.cacheMu.Lock()
	t.cache = t.evalList()
	t.cacheMu.Unlock()
	for _ = range t.cacheTicker.C {
		t.cacheMu.Lock()
		t.cache = t.evalList()
		t.cacheMu.Unlock()
	}
}

// evalList constructs an actual target list for calls to List(), as well as
// updates t.nameToIP. This list can be provided directly to calls to List, or
// to the cache.
// evalList is responsible for filtering out stale entries in an RTC List.  If
// the majority of elements are outdated, it is likely a critical error has
// occurred with RTC. In this case (if caching was enabled), we will fall back
// on the last known-good list.
func (t *Targets) evalList() []string {
	resp, err := t.rtc.List()
	if err != nil {
		t.l.Errorf("rtc.evalList() : unable to produce RTC list : %v", err)
		return nil
	}

	var targs []string
	cutoff := timeNow().Add(-t.expire)
	staleCnt := 0
	for _, v := range resp {
		// Filter out old entries, as configured by "exp_msec".
		updateTime, err := time.Parse(time.RFC3339Nano, v.UpdateTime)
		if err != nil {
			t.l.Errorf("rtc.evalList: unable to parse variable UpdateTime (was %v): %v", v.UpdateTime, err)
			continue
		}
		if updateTime.Before(cutoff) {
			staleCnt++
			continue
		}

		// Once stale entries are filtered, parse the variable
		ta, err := t.targsFromVar(v)
		if err != nil {
			t.l.Errorf("rtc.evalList() : skipping variable %v : %v", v.Name, err)
			continue
		}
		if ta != "" {
			targs = append(targs, ta)
		}
	}

	// Check ratio of stale variables. If too many were flagged as stale, return
	// the old cache.
	if float64(staleCnt)/float64(len(resp)) >= .5 {
		t.l.Warningf("rtc.evalList(): %f%% of entries are stale", float64(staleCnt)*100.0/float64(len(resp)))
		t.cacheMu.RLock()
		defer t.cacheMu.RUnlock()
		if t.cache != nil {
			t.l.Warningf("rtc.evalList(): falling back to cache due to stale entire")
			return t.cache
		}
	}

	return targs
}

// targsFromVar produces the target associated with a variable in an rtcservice.Config,
// after making the appropriate filters. While doing this, t.nameToIP will be
// updated.
func (t *Targets) targsFromVar(v *runtimeconfig.Variable) (string, error) {
	// Get the Value
	val, err := t.rtc.Val(v)
	if err != nil {
		return "", err
	}

	pb := &rtcreporter.RtcTargetInfo{}
	if err := proto.Unmarshal(val, pb); err != nil {
		return "", fmt.Errorf("rtc.targsFromVar: Unable to unmarshal proto : %v", err)
	}

	// Filter by group
	ingroup := len(t.groups) == 0
	for _, g := range pb.GetGroups() {
		ingroup = ingroup || t.groups[g]
	}
	if !ingroup {
		return "", nil
	}

	// Get name names will be of the form "/some/path/from/config/<key>"
	vparts := strings.Split(v.Name, "/")
	targ := vparts[len(vparts)-1]

	// Filter by addrTag. We store a list in order to check if multiple matches
	// found.
	addrs := []string{}
	for _, a := range pb.GetAddresses() {
		if t.addrTag == a.GetTag() {
			addrs = append(addrs, a.GetAddress())
		}
	}

	if len(addrs) != 1 {
		return "", fmt.Errorf("rtc.targsFromVar(): Got %v addrs, want 1", len(addrs))
	}
	ip := net.ParseIP(addrs[0])
	if ip == nil {
		return "", fmt.Errorf("rtc.targsFromVar(): Failed to parse IP %q", addrs[0])
	}
	t.nameToIPMu.Lock()
	t.nameToIP[targ] = ip
	t.nameToIPMu.Unlock()

	return targ, nil
}

// New returns an rtc resolver / lister, given a defining protobuf.
func New(pb *TargetsConf, proj string, l *logger.Logger) (*Targets, error) {
	rtc, err := rtcservice.New(proj, pb.GetCfg(), nil)
	if err != nil {
		err = fmt.Errorf("newRTC: Error building rtc client %v for targets: %v", pb.GetCfg(), err)
		return nil, err
	}

	expire := time.Duration(pb.GetExpireMsec()) * time.Millisecond

	gs := make(map[string]bool)
	for _, g := range pb.GetGroups() {
		gs[g] = true
	}

	t := &Targets{
		rtc:      rtc,
		l:        l,
		expire:   expire,
		groups:   gs,
		addrTag:  pb.GetResolveTag(),
		nameToIP: make(map[string]net.IP)}

	// A positive re_eval_sec will yield a non-nil ticker, enabling the List
	// cache. List cache should only be enabled once t.l and t.r has been set.
	if pb.GetReEvalSec() > 0 {
		t.cacheTicker = time.NewTicker(time.Duration(pb.GetReEvalSec()) * time.Second)
		go t.runc()
	}

	return t, nil
}
