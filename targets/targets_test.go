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

package targets_test

import (
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/targets/testdata"
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

type mockLDLister struct {
	list []string
}

func (mldl *mockLDLister) List() ([]string, error) {
	return mldl.list, nil
}

// TestList does not test the targets.New function, and is specifically testing
// the implementation of targets.targets directly
func TestList(t *testing.T) {
	var rows = []struct {
		hosts  []string
		re     string
		ldList []string
		expect []string
	}{
		{
			[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			"",
			[]string{"hostB"}, // hostB is lameduck.
			[]string{"www.google.com", "127.0.0.1", "hostA", "hostC"},
		},
		{
			[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			".*",
			nil,
			[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
		},
		{
			[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			"host.*",
			[]string{"hostC"}, // hostC is lameduck.
			[]string{"hostA", "hostB"},
		},
		{
			[]string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			"empty.*",
			nil,
			[]string{},
		},
	}

	for id, r := range rows {
		targetsDef := &TargetsDef{
			Regex: proto.String(r.re),
			Type: &TargetsDef_HostNames{
				HostNames: strings.Join(r.hosts, ","),
			},
		}
		bt, err := targets.New(targetsDef, &mockLDLister{r.ldList}, nil, nil, nil)
		if err != nil {
			t.Fatalf("Unexpected error building targets: %v", err)
		}
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
	tgts, err := targets.New(targetsDef, nil, nil, nil, l)
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
	tgts := targets.StaticTargets(testHosts)
	if !reflect.DeepEqual(tgts.List(), strings.Split(testHosts, ",")) {
		t.Errorf("StaticTargets not working as expected. Got list: %q, Expected: %s", tgts.List(), strings.Split(testHosts, ","))
	}
}

type testTargetsType struct {
	names []string
}

func (tgts *testTargetsType) List() []string {
	return tgts.names
}

func (tgts *testTargetsType) Resolve(name string, ipVer int) (net.IP, error) {
	return nil, errors.New("resolve not implemented")
}

func TestGetExtensionTargets(t *testing.T) {
	targetsDef := &TargetsDef{}

	// This has the same effect as using the following in your config:
	// targets {
	//    [cloudprober.targets.testdata.fancy_targets] {
	//      name: "fancy"
	//    }
	// }
	err := proto.SetExtension(targetsDef, testdata.E_FancyTargets, &testdata.FancyTargets{Name: proto.String("fancy")})
	if err != nil {
		t.Fatalf("error setting up extension in test targets proto: %v", err)
	}
	tgts, err := targets.New(targetsDef, nil, nil, nil, nil)
	if err == nil {
		t.Errorf("Expected error in building targets from extensions, got nil. targets: %v", tgts)
	}
	testTargets := []string{"a", "b"}
	targets.RegisterTargetsType(200, func(conf interface{}, l *logger.Logger) (targets.Targets, error) {
		return &testTargetsType{names: testTargets}, nil
	})
	tgts, err = targets.New(targetsDef, nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Got error in building targets from extensions: %v.", err)
	}
	tgtsList := tgts.List()
	if !reflect.DeepEqual(tgtsList, testTargets) {
		t.Errorf("Extended targets: tgts.List()=%v, expected=%v", tgtsList, testTargets)
	}
}
