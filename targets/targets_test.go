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

package targets

import (
	"reflect"
	"strings"
	"testing"

	"github.com/google/cloudprober/logger"
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

// TestList does not test the targets.New function, and is specifically testing
// the implementation of targets.targets directly
// TODO: Cannot test lameduck until it is mockable.
func TestList(t *testing.T) {
	var rows = []struct {
		hosts  []string
		re     string
		expect []string
	}{
		{[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			"",
			[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"}},
		{[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			".*",
			[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"}},
		{[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			"host.*",
			[]string{"hostA", "hostB", "hostC"}},
		{[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			"empty.*",
			[]string{}},
	}

	for id, r := range rows {
		bt, err := baseTargets(nil, r.re, true)
		if err != nil {
			t.Fatal("Unexpected error building baseTarget: ", err)
		}
		bt.l = &staticLister{list: r.hosts}
		got := bt.List()

		// Got \subset Expected
		missing := getMissing(got, r.expect)
		if len(missing) != 0 {
			t.Error("In test row ", id, ": Got unexpected hosts: ", missing)
		}
		// Expected \subset Got
		missing = getMissing(r.expect, got)
		if len(missing) != 0 {
			t.Error("In test row ", id, ": Expected hosts: ", missing)
		}
	}
}

func TestDummyTargets(t *testing.T) {
	targetsDef := &TargetsDef{
		Type: &TargetsDef_DummyTargets{
			DummyTargets: &DummyTargets{},
		},
	}
	l := &logger.Logger{}
	tgts, err := New(targetsDef, nil, nil, l)
	if err != nil {
		t.Fatalf("targets.New(...) Unexpected errors %v", err)
	}
	got := tgts.List()
	want := []string{""}
	if !reflect.DeepEqual(got, []string{""}) {
		t.Errorf("tgts.List() = %q, want %q", got, want)
	}
	ip, err := tgts.Resolve(got[0], 4)
	if err != nil {
		t.Errorf("tgts.Resolve(%q, 4) Unexpected errors %v", got[0], err)
	} else if !ip.IsUnspecified() {
		t.Errorf("tgts.Resolve(%q, 4) = %v is specified, expected unspecified", got[0], ip)
	}
	ip, err = tgts.Resolve(got[0], 6)
	if err != nil {
		t.Errorf("tgts.Resolve(%q, 6) Unexpected errors %v", got[0], err)
	} else if !ip.IsUnspecified() {
		t.Errorf("tgts.Resolve(%q, 6) = %v is specified, expected unspecified", got[0], ip)
	}
}

func TestStaticTargets(t *testing.T) {
	testHosts := "host1,host2"
	tgts := StaticTargets(testHosts)
	if !reflect.DeepEqual(tgts.List(), strings.Split(testHosts, ",")) {
		t.Errorf("StaticTargets not working as expected. Got list: %q, Expected: %s", tgts.List(), strings.Split(testHosts, ","))
	}
}
