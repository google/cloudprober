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

package targets

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	rdsclientpb "github.com/google/cloudprober/rds/client/proto"
	"github.com/google/cloudprober/targets/endpoint"
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

type mockLister struct {
	list []endpoint.Endpoint
}

func (mldl *mockLister) ListEndpoints() []endpoint.Endpoint {
	return mldl.list
}

// TestList does not test the New function, and is specifically testing
// the implementation of targets directly
func TestList(t *testing.T) {
	var rows = []struct {
		desc   string
		hosts  []string
		re     string
		ldList []endpoint.Endpoint
		expect []string
	}{
		{
			desc:   "hostB is lameduck",
			hosts:  []string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			ldList: endpoint.EndpointsFromNames([]string{"hostB"}), // hostB is lameduck.
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
			ldList: endpoint.EndpointsFromNames([]string{"hostC"}), // hostC is lameduck.
			expect: []string{"hostA", "hostB"},
		},
		{
			desc:   "only hosts starting with host and hostC was lameducked before hostC was updated",
			hosts:  []string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			re:     "host.*",
			ldList: []endpoint.Endpoint{{Name: "hostC", LastUpdated: time.Now().Add(-time.Hour)}}, // hostC is lameduck.
			expect: []string{"hostA", "hostB", "hostC"},
		},
		{
			desc:  "empty as no hosts match the regex",
			hosts: []string{"www.google.com", "127.0.0.1", "hostA", "hostB", "hostC"},
			re:    "empty.*",
		},
	}

	for _, r := range rows {
		baseTime := time.Now()

		var targetEP []endpoint.Endpoint
		for _, host := range r.hosts {
			targetEP = append(targetEP, endpoint.Endpoint{
				Name:        host,
				LastUpdated: baseTime,
			})
		}

		t.Run(r.desc, func(t *testing.T) {
			bt, err := baseTargets(nil, &mockLister{r.ldList}, nil)
			if err != nil {
				t.Errorf("Unexpected error building targets: %v", err)
				return
			}
			bt.re = regexp.MustCompile(r.re)
			bt.lister = &mockLister{targetEP}

			got := endpoint.NamesFromEndpoints(bt.ListEndpoints())
			if !reflect.DeepEqual(got, r.expect) {
				// Ignore the case when both slices are zero length, DeepEqual doesn't
				// handle initialized but zero and non-initialized comparison very well.
				if !(len(got) == 0 && len(r.expect) == 0) {
					t.Errorf("tgts.List(): got=%v, want=%v", got, r.expect)
				}
			}

			gotEndpoints := endpoint.NamesFromEndpoints(bt.ListEndpoints())
			if !reflect.DeepEqual(gotEndpoints, r.expect) {
				// Ignore the case when both slices are zero length, DeepEqual doesn't
				// handle initialized but zero and non-initialized comparison very well.
				if !(len(got) == 0 && len(r.expect) == 0) {
					t.Errorf("tgts.ListEndpoints(): got=%v, want=%v", gotEndpoints, r.expect)
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
	tgts, err := New(targetsDef, nil, nil, nil, l)
	if err != nil {
		t.Fatalf("New(...) Unexpected errors %v", err)
	}
	got := endpoint.NamesFromEndpoints(tgts.ListEndpoints())
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
	got := endpoint.NamesFromEndpoints(StaticTargets(testHosts).ListEndpoints())
	if !reflect.DeepEqual(got, strings.Split(testHosts, ",")) {
		t.Errorf("StaticTargets not working as expected. Got list: %q, Expected: %s", got, strings.Split(testHosts, ","))
	}
}

type testTargetsType struct {
	names []string
}

func (tgts *testTargetsType) List() []string {
	return tgts.names
}

func (tgts *testTargetsType) ListEndpoints() []endpoint.Endpoint {
	return endpoint.EndpointsFromNames(tgts.names)
}

func (tgts *testTargetsType) Resolve(name string, ipVer int) (net.IP, error) {
	return nil, errors.New("resolve not implemented")
}

func TestGetExtensionTargets(t *testing.T) {
	targetsDef := &targetspb.TargetsDef{}

	// This has the same effect as using the following in your config:
	// targets {
	//    [cloudprober.testdata.fancy_targets] {
	//      name: "fancy"
	//    }
	// }
	err := proto.SetExtension(targetsDef, testdatapb.E_FancyTargets, &testdatapb.FancyTargets{Name: proto.String("fancy")})
	if err != nil {
		t.Fatalf("error setting up extension in test targets proto: %v", err)
	}
	tgts, err := New(targetsDef, nil, nil, nil, nil)
	if err == nil {
		t.Errorf("Expected error in building targets from extensions, got nil. targets: %v", tgts)
	}
	testTargets := []string{"a", "b"}
	RegisterTargetsType(200, func(conf interface{}, l *logger.Logger) (Targets, error) {
		return &testTargetsType{names: testTargets}, nil
	})
	tgts, err = New(targetsDef, nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Got error in building targets from extensions: %v.", err)
	}
	tgtsList := endpoint.NamesFromEndpoints(tgts.ListEndpoints())
	if !reflect.DeepEqual(tgtsList, testTargets) {
		t.Errorf("Extended targets: tgts.List()=%v, expected=%v", tgtsList, testTargets)
	}
}

func TestSharedTargets(t *testing.T) {
	testHosts := []string{"host1", "host2"}

	// Create shared targets and re-use them.
	SetSharedTargets("shared_test_targets", StaticTargets(strings.Join(testHosts, ",")))

	var tgts [2]Targets

	for i := range tgts {
		targetsDef := &targetspb.TargetsDef{
			Type: &targetspb.TargetsDef_SharedTargets{SharedTargets: "shared_test_targets"},
		}

		var err error
		tgts[i], err = New(targetsDef, nil, nil, nil, nil)

		if err != nil {
			t.Errorf("got error while creating targets from shared targets: %v", err)
		}

		got := endpoint.NamesFromEndpoints(tgts[i].ListEndpoints())
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
				pb.RdsServerOptions = &rdsclientpb.ClientConf_ServerOptions{
					ServerAddress: proto.String(r.localAddr),
				}
			}

			globalOpts := &targetspb.GlobalTargetsOptions{}
			if r.globalAddr != "" {
				globalOpts.RdsServerOptions = &rdsclientpb.ClientConf_ServerOptions{
					ServerAddress: proto.String(r.globalAddr),
				}
			}

			_, cc, err := RDSClientConf(pb, globalOpts, nil)
			if (err != nil) != r.wantErr {
				t.Errorf("wantErr: %v, got err: %v", r.wantErr, err)
			}

			if err != nil {
				return
			}

			if cc.GetServerOptions().GetServerAddress() != r.wantAddr {
				t.Errorf("Got RDS server address: %s, wanted: %s", cc.GetServerOptions().GetServerAddress(), r.wantAddr)
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
