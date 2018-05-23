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

package rtc

import (
	"encoding/base64"
	"net"
	"sort"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	cpb "github.com/google/cloudprober/targets/rtc/rtcreporter/proto"
	"github.com/google/cloudprober/targets/rtc/rtcservice"
	"github.com/kylelemons/godebug/pretty"
)

// getMissing returns a list of items in "elems" missing from "from". Cannot
// handle duplicate elements.
func getMissing(elems []string, from []string) []string {
	var missing []string
	set := make(map[string]bool, len(from))
	for _, e := range from {
		set[e] = true
	}

	for _, e := range elems {
		if !set[e] {
			missing = append(missing, e)
		}
	}
	return missing
}

type targInfo struct {
	value      *cpb.RtcTargetInfo
	updatetime string
}

type addr struct {
	tag  string
	addr string
}

func mkTargetInfo(name string, tags []string, addrs []addr) *cpb.RtcTargetInfo {
	pb := &cpb.RtcTargetInfo{
		InstanceName: proto.String(name),
		Groups:       tags,
	}
	addresses := make([]*cpb.RtcTargetInfo_Address, len(addrs))
	for i, a := range addrs {
		addresses[i] = &cpb.RtcTargetInfo_Address{Tag: proto.String(a.tag), Address: proto.String(a.addr)}
	}
	pb.Addresses = addresses

	return pb
}

// Tests resolving with RTC works as expected.
func TestRtctargsResolve(t *testing.T) {
	const (
		// Now
		t0 = "2016-12-06T02:43:40.558291428Z"
		// Now - 0.5s
		t1 = "2016-12-06T02:43:40.058291428Z"
		// Now - 1s
		t2 = "2016-12-06T02:43:39.558291428Z"
		// Now - 1.5s
		t3 = "2016-12-06T02:43:39.058291428Z"
	)
	t0t, err := time.Parse(time.RFC3339Nano, t0)
	if err != nil {
		t.Fatal("Unable to parse time 0: ", err)
	}
	// Sets a static time for calls to timeNow()
	timeNow = func() time.Time { return t0t }
	l := &logger.Logger{}

	var rows = []struct {
		name      string
		vars      []targInfo
		resAddr   string
		groups    map[string]bool
		want      map[string]string
		wantError bool
	}{
		{"Test normal usage",
			[]targInfo{
				{
					value: mkTargetInfo("host1", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"},
						addr{tag: "ip2", addr: "1.1.1.2"},
						addr{tag: "ip3", addr: "1.1.1.3"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host2", []string{}, []addr{
						addr{tag: "ip1", addr: "2.2.2.1"},
						addr{tag: "ip2", addr: "2.2.2.2"},
						addr{tag: "ip3", addr: "2.2.2.3"}}),
					updatetime: t0}},
			"ip1", map[string]bool{},
			map[string]string{"host1": "1.1.1.1", "host2": "2.2.2.1"}, false},
		{"Test group filter: single filter, single group",
			[]targInfo{
				{
					value: mkTargetInfo("host1", []string{"g1"}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"},
						addr{tag: "ip2", addr: "1.1.1.2"},
						addr{tag: "ip3", addr: "1.1.1.3"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host2", []string{"g2"}, []addr{
						addr{tag: "ip1", addr: "2.2.2.1"},
						addr{tag: "ip2", addr: "2.2.2.2"},
						addr{tag: "ip3", addr: "2.2.2.3"}}),
					updatetime: t0}},
			"ip1", map[string]bool{"g1": true},
			map[string]string{"host1": "1.1.1.1"}, false},
		{"Test group filter: multi filter, single group",
			[]targInfo{
				{
					value: mkTargetInfo("host1", []string{"g1"}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"},
						addr{tag: "ip2", addr: "1.1.1.2"},
						addr{tag: "ip3", addr: "1.1.1.3"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host2", []string{"g2"}, []addr{
						addr{tag: "ip1", addr: "2.2.2.1"},
						addr{tag: "ip2", addr: "2.2.2.2"},
						addr{tag: "ip3", addr: "2.2.2.3"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host3", []string{"g3"}, []addr{
						addr{tag: "ip1", addr: "3.3.3.1"},
						addr{tag: "ip2", addr: "3.3.3.2"},
						addr{tag: "ip3", addr: "3.3.3.3"}}),
					updatetime: t0}},
			"ip1", map[string]bool{"g1": true, "g2": true},
			map[string]string{"host1": "1.1.1.1", "host2": "2.2.2.1"}, false},
		{"Test group filter: single filter, multi group",
			[]targInfo{
				{
					value: mkTargetInfo("host1", []string{"g1", "g2"}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"},
						addr{tag: "ip2", addr: "1.1.1.2"},
						addr{tag: "ip3", addr: "1.1.1.3"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host2", []string{"g2", "g3"}, []addr{
						addr{tag: "ip1", addr: "2.2.2.1"},
						addr{tag: "ip2", addr: "2.2.2.2"},
						addr{tag: "ip3", addr: "2.2.2.3"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host3", []string{"g3", "g4"}, []addr{
						addr{tag: "ip1", addr: "3.3.3.1"},
						addr{tag: "ip2", addr: "3.3.3.2"},
						addr{tag: "ip3", addr: "3.3.3.3"}}),
					updatetime: t0}},
			"ip1", map[string]bool{"g2": true},
			map[string]string{"host1": "1.1.1.1", "host2": "2.2.2.1"}, false},
		{"Test group filter: multi filter, multi group",
			[]targInfo{
				{
					value: mkTargetInfo("host1", []string{"g1", "g2"}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"},
						addr{tag: "ip2", addr: "1.1.1.2"},
						addr{tag: "ip3", addr: "1.1.1.3"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host2", []string{"g2", "g3"}, []addr{
						addr{tag: "ip1", addr: "2.2.2.1"},
						addr{tag: "ip2", addr: "2.2.2.2"},
						addr{tag: "ip3", addr: "2.2.2.3"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host3", []string{"g3", "g4"}, []addr{
						addr{tag: "ip1", addr: "3.3.3.1"},
						addr{tag: "ip2", addr: "3.3.3.2"},
						addr{tag: "ip3", addr: "3.3.3.3"}}),
					updatetime: t0}},
			"ip1", map[string]bool{"g1": true, "g4": true},
			map[string]string{"host1": "1.1.1.1", "host3": "3.3.3.1"}, false},
		{"Test time filter normal usage",
			[]targInfo{
				{
					value: mkTargetInfo("host1", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"},
						addr{tag: "ip2", addr: "1.1.1.2"},
						addr{tag: "ip3", addr: "1.1.1.3"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host2", []string{}, []addr{
						addr{tag: "ip1", addr: "2.2.2.1"},
						addr{tag: "ip2", addr: "2.2.2.2"},
						addr{tag: "ip3", addr: "2.2.2.3"}}),
					updatetime: t3}},
			"ip1", map[string]bool{},
			map[string]string{"host1": "1.1.1.1"}, false},
	}

	// For every row, setup the experiment, and see if listing returns r.want. If
	// test setup fails, we skip to next row.
Nextrow:
	for id, r := range rows {
		// Setup targs
		rtc := rtcservice.NewStub()
		for _, v := range r.vars {
			data, err := proto.Marshal(v.value)
			if err != nil {
				t.Errorf("In test row %v : %v : Unable to marshal pb %v", id, r.name, v.value)
				continue Nextrow
			}
			encoded := base64.StdEncoding.EncodeToString(data)
			rtc.WriteTime(v.value.GetInstanceName(), encoded, v.updatetime)
		}
		targs := &Targets{
			rtc:      rtc,
			expire:   time.Second,
			l:        l,
			groups:   r.groups,
			addrTag:  r.resAddr,
			nameToIP: make(map[string]net.IP),
		}
		// List targets
		gotlist := targs.List()
		// TODO(izzycecil): Need to catch errors with something like this. This
		// requires looking at the logger.
		// if (err != nil) != r.wantError {
		// 	t.Errorf("%v: targs.List() gave error %q. r.wantError = %v", r.name, err, r.wantError)
		// 	continue Nextrow
		// }
		// if err != nil {
		// 	continue Nextrow
		// }
		// Make resolutions
		got := make(map[string]string)
		for _, l := range gotlist {
			ip, err := targs.Resolve(l, 4)
			if err != nil {
				t.Errorf("%v: targs.Resolve(%q, 4) gave error %q", r.name, l, err)
			}
			got[l] = ip.String()
		}
		if diff := pretty.Compare(r.want, got); diff != "" {
			t.Errorf("%v : got diff:\n%s", r.name, diff)
		}
	}
}

func TestRtctargsStale(t *testing.T) {
	const (
		// Now
		t0 = "2016-12-06T02:43:40.558291428Z"
		// Now - 0.5s
		t1 = "2016-12-06T02:43:40.058291428Z"
		// Now - 1s
		t2 = "2016-12-06T02:43:39.558291428Z"
		// Now - 1.5s
		t3 = "2016-12-06T02:43:39.058291428Z"
	)
	t0t, err := time.Parse(time.RFC3339Nano, t0)
	if err != nil {
		t.Fatal("Unable to parse time 0: ", err)
	}
	// Sets a static time for calls to timeNow()
	timeNow = func() time.Time { return t0t }
	l := &logger.Logger{}

	var rows = []struct {
		name    string
		vars    []targInfo
		resAddr string
		cache   []string
		want    []string
	}{
		{"All up to date",
			[]targInfo{
				{
					value: mkTargetInfo("host1", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host2", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host3", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host4", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t0}},
			"ip1", []string{"cache"},
			[]string{"host1", "host2", "host3", "host4"}},
		{"50% down, with cache",
			[]targInfo{
				{
					value: mkTargetInfo("host1", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t3},
				{
					value: mkTargetInfo("host2", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t3},
				{
					value: mkTargetInfo("host3", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host4", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t0}},
			"ip1", []string{"cache"},
			[]string{"cache"}},
		{"50% down, with no cache",
			[]targInfo{
				{
					value: mkTargetInfo("host1", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t3},
				{
					value: mkTargetInfo("host2", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t3},
				{
					value: mkTargetInfo("host3", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t0},
				{
					value: mkTargetInfo("host4", []string{}, []addr{
						addr{tag: "ip1", addr: "1.1.1.1"}}),
					updatetime: t0}},
			"ip1", nil,
			[]string{"host3", "host4"}},
	}

	// For every row, setup the experiment, and see if listing returns r.want. If
	// test setup fails, we skip to next row.
Nextrow:
	for id, r := range rows {
		// Setup targs
		rtc := rtcservice.NewStub()
		for _, v := range r.vars {
			data, err := proto.Marshal(v.value)
			if err != nil {
				t.Errorf("In test row %v : %v : Unable to marshal pb %v", id, r.name, v.value)
				continue Nextrow
			}
			encoded := base64.StdEncoding.EncodeToString(data)
			rtc.WriteTime(v.value.GetInstanceName(), encoded, v.updatetime)
		}
		targs := &Targets{
			rtc:      rtc,
			expire:   time.Second,
			l:        l,
			addrTag:  r.resAddr,
			cache:    r.cache,
			nameToIP: make(map[string]net.IP),
		}
		// List targets
		got := targs.List()
		sort.Strings(got)
		sort.Strings(r.want)

		if diff := pretty.Compare(r.want, got); diff != "" {
			t.Errorf("%v : got diff:\n%s", r.name, diff)
		}
	}
}
