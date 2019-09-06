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
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets"
	targetspb "github.com/google/cloudprober/targets/proto"
	testdatapb "github.com/google/cloudprober/targets/testdata"
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

func (mldl *mockLDLister) List() []string {
	return mldl.list
}

func TestEndpointsFromNames(t *testing.T) {
	names := []string{"targetA", "targetB", "targetC"}
	endpoints := targets.EndpointsFromNames(names)

	for i := range names {
		ep := endpoints[i]

		if ep.Name != names[i] {
			t.Errorf("Endpoint.Name=%s, want=%s", ep.Name, names[i])
		}
		if ep.Port != 0 {
			t.Errorf("Endpoint.Port=%d, want=0", ep.Port)
		}
		if len(ep.Labels) != 0 {
			t.Errorf("Endpoint.Labels=%v, want={}", ep.Labels)
		}
	}
}

// TestList does not test the targets.New function, and is specifically testing
// the implementation of targets.targets directly
func TestList(t *testing.T) {
	var rows = []struct {
		desc   string
		hosts  []string
		re     string
		ldList []string
		expect []string
	}{
		{
			desc:   "hostB is lameduck",
			hosts:  []string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			ldList: []string{"hostB"}, // hostB is lameduck.
			expect: []string{"www.google.com", "127.0.0.1", "hostA", "hostC"},
		},
		{
			desc:   "all hosts no lameduck",
			hosts:  []string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			re:     ".*",
			expect: []string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
		},
		{
			desc:   "only hosts starting with host and hostC is lameduck",
			hosts:  []string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			re:     "host.*",
			ldList: []string{"hostC"}, // hostC is lameduck.
			expect: []string{"hostA", "hostB"},
		},
		{
			desc:  "empty as no hosts match the regex",
			hosts: []string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			re:    "empty.*",
		},
	}

	for _, r := range rows {
		targetsDef := &targetspb.TargetsDef{
			Regex: proto.String(r.re),
			Type: &targetspb.TargetsDef_HostNames{
				HostNames: strings.Join(r.hosts, ","),
			},
		}

		t.Run(r.desc, func(t *testing.T) {
			bt, err := targets.New(targetsDef, &mockLDLister{r.ldList}, nil, nil, nil)
			if err != nil {
				t.Errorf("Unexpected error building targets: %v", err)
				return
			}

			got := bt.List()
			if !reflect.DeepEqual(got, r.expect) {
				// Ignore the case when both slices are zero length, DeepEqual doesn't
				// handle initialized but zero and non-initialized comparison very well.
				if !(len(got) == 0 && len(r.expect) == 0) {
					t.Errorf("tgts.List(): got=%v, want=%v", got, r.expect)
				}
			}

			gotEndpoints := bt.ListEndpoints()
			wantEndpoints := targets.EndpointsFromNames(r.expect)
			if !reflect.DeepEqual(gotEndpoints, wantEndpoints) {
				// Ignore the case when both slices are zero length, DeepEqual doesn't
				// handle initialized but zero and non-initialized comparison very well.
				if !(len(got) == 0 && len(r.expect) == 0) {
					t.Errorf("tgts.ListEndpoints(): got=%v, want=%v", gotEndpoints, wantEndpoints)
				}
			}

		})
	}
}

func TestDummyTargets(t *testing.T) {
	targetsDef := &targetspb.TargetsDef{
		Type: &targetspb.TargetsDef_DummyTargets{
			DummyTargets: &targetspb.DummyTargets{},
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

func (tgts *testTargetsType) ListEndpoints() []targets.Endpoint {
	return targets.EndpointsFromNames(tgts.names)
}

func (tgts *testTargetsType) Resolve(name string, ipVer int) (net.IP, error) {
	return nil, errors.New("resolve not implemented")
}

func TestGetExtensionTargets(t *testing.T) {
	targetsDef := &targetspb.TargetsDef{}

	// This has the same effect as using the following in your config:
	// targets {
	//    [cloudprober.targets.testdata.fancy_targets] {
	//      name: "fancy"
	//    }
	// }
	err := proto.SetExtension(targetsDef, testdatapb.E_FancyTargets, &testdatapb.FancyTargets{Name: proto.String("fancy")})
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

func TestSharedTargets(t *testing.T) {
	testHosts := []string{"host1", "host2"}

	// Create shared targets and re-use them.
	targets.SetSharedTargets("shared_test_targets", targets.StaticTargets(strings.Join(testHosts, ",")))

	var tgts [2]targets.Targets

	for i := range tgts {
		targetsDef := &targetspb.TargetsDef{
			Type: &targetspb.TargetsDef_SharedTargets{SharedTargets: "shared_test_targets"},
		}

		var err error
		tgts[i], err = targets.New(targetsDef, nil, nil, nil, nil)

		if err != nil {
			t.Errorf("got error while creating targets from shared targets: %v", err)
		}

		got := tgts[i].List()
		if !reflect.DeepEqual(got, testHosts) {
			t.Errorf("Unexpected targets: tgts.List()=%v, expected=%v", got, testHosts)
		}
	}
}

func TestRDSClientConf(t *testing.T) {
	provider := "test-provider"
	rPath := "test-rsources"

	var rows = []struct {
		desc       string
		localAddr  string
		globalAddr string
		provider   string
		wantErr    bool
		wantAddr   string
	}{
		{
			desc:     "Error as RDS server address is not initialized",
			provider: provider,
			wantErr:  true,
		},
		{
			desc:       "Pick global address",
			provider:   provider,
			globalAddr: "test-global-addr",
			wantAddr:   "test-global-addr",
		},
		{
			desc:       "Pick local address over global",
			provider:   provider,
			localAddr:  "test-local-addr",
			globalAddr: "test-global-addr",
			wantAddr:   "test-local-addr",
		},
		{
			desc:      "Error because no provider",
			provider:  "",
			localAddr: "test-local-addr",
			wantAddr:  "test-local-addr",
			wantErr:   true,
		},
	}

	for _, r := range rows {
		t.Run(r.desc, func(t *testing.T) {
			pb := &targetspb.RDSTargets{
				ResourcePath: proto.String(fmt.Sprintf("%s://%s", r.provider, rPath)),
			}
			if r.localAddr != "" {
				pb.RdsServerAddress = proto.String(r.localAddr)
			}

			globalOpts := &targetspb.GlobalTargetsOptions{}
			if r.globalAddr != "" {
				globalOpts.RdsServerAddress = proto.String(r.globalAddr)
			}

			_, cc, err := targets.RDSClientConf(pb, globalOpts)
			if (err != nil) != r.wantErr {
				t.Errorf("wantErr: %v, got err: %v", r.wantErr, err)
			}

			if err != nil {
				return
			}

			if cc.GetServerAddr() != r.wantAddr {
				t.Errorf("Got RDS server address: %s, wanted: %s", cc.GetServerAddr(), r.wantAddr)
			}
			if cc.GetRequest().GetProvider() != provider {
				t.Errorf("Got provider: %s, wanted: %s", cc.GetRequest().GetProvider(), provider)
			}
			if cc.GetRequest().GetResourcePath() != rPath {
				t.Errorf("Got resource path: %s, wanted: %s", cc.GetRequest().GetResourcePath(), rPath)
			}
		})
	}
}
