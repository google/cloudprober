// Copyright 2021 The Cloudprober Authors.
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

package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	configpb "github.com/google/cloudprober/rds/file/proto"
	rdspb "github.com/google/cloudprober/rds/proto"
	"google.golang.org/protobuf/proto"
)

var testResourcesFiles = map[string][]string{
	"textpb": []string{"testdata/targets1.textpb", "testdata/targets2.textpb"},
	"json":   []string{"testdata/targets.json"},
}

var testExpectedResources = []*rdspb.Resource{
	{
		Name: proto.String("switch-xx-1"),
		Port: proto.Int32(8080),
		Ip:   proto.String("10.1.1.1"),
		Labels: map[string]string{
			"device_type": "switch",
			"cluster":     "xx",
		},
	},
	{
		Name: proto.String("switch-xx-2"),
		Port: proto.Int32(8081),
		Ip:   proto.String("10.1.1.2"),
		Labels: map[string]string{
			"cluster": "xx",
		},
	},
	{
		Name: proto.String("switch-yy-1"),
		Port: proto.Int32(8080),
		Ip:   proto.String("10.1.2.1"),
	},
	{
		Name: proto.String("switch-zz-1"),
		Port: proto.Int32(8080),
		Ip:   proto.String("::aaa:1"),
	},
}

func compareResourceList(t *testing.T, got []*rdspb.Resource, want []*rdspb.Resource) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("Got resources: %d, expected: %d", len(got), len(want))
	}
	for i := range want {
		if got[i].String() != want[i].String() {
			t.Errorf("ListResources: got[%d]:\n%s\nexpected[%d]:\n%s", i, got[i].String(), i, want[i].String())
		}
	}
}

func TestListResources(t *testing.T) {
	for _, filetype := range []string{"textpb", "json"} {
		t.Run(filetype, func(t *testing.T) {
			p, err := New(&configpb.ProviderConfig{FilePath: testResourcesFiles[filetype]}, nil)
			if err != nil {
				t.Fatalf("Unexpected error while creating new provider: %v", err)
			}

			for _, test := range []struct {
				desc          string
				resourcePath  string
				f             []*rdspb.Filter
				wantResources []*rdspb.Resource
			}{
				{
					desc:          "no_filter",
					wantResources: testExpectedResources,
				},
				{
					desc: "with_filter",
					f: []*rdspb.Filter{
						{
							Key:   proto.String("labels.cluster"),
							Value: proto.String("xx"),
						},
					},
					wantResources: testExpectedResources[:2],
				},
			} {
				t.Run(test.desc, func(t *testing.T) {
					got, err := p.ListResources(&rdspb.ListResourcesRequest{Filter: test.f})
					if err != nil {
						t.Fatalf("Unexpected error while listing resources: %v", err)
					}
					compareResourceList(t, got.Resources, test.wantResources)
				})
			}
		})
	}
}

func TestListResourcesWithResourcePath(t *testing.T) {
	p, err := New(&configpb.ProviderConfig{FilePath: testResourcesFiles["textpb"]}, nil)
	if err != nil {
		t.Fatalf("Unexpected error while creating new provider: %v", err)
	}
	got, err := p.ListResources(&rdspb.ListResourcesRequest{ResourcePath: proto.String(testResourcesFiles["textpb"][1])})
	if err != nil {
		t.Fatalf("Unexpected error while listing resources: %v", err)
	}
	compareResourceList(t, got.Resources, testExpectedResources[2:])
}

func BenchmarkListResources(b *testing.B) {
	for _, n := range []int{100, 10000, 1000000} {
		for _, filters := range [][]*rdspb.Filter{nil, []*rdspb.Filter{{Key: proto.String("name"), Value: proto.String("host-1.*")}}} {
			b.Run(fmt.Sprintf("%d-resources,%d-filters", n, len(filters)), func(b *testing.B) {
				b.StopTimer()
				ls := &lister{
					resources: make([]*rdspb.Resource, n),
				}
				for i := 0; i < n; i++ {
					ls.resources[i] = &rdspb.Resource{
						Name: proto.String(fmt.Sprintf("host-%d", i)),
						Ip:   proto.String("10.1.1.1"),
						Port: proto.Int32(80),
						Labels: map[string]string{
							"index": strconv.Itoa(i),
						},
						LastUpdated: proto.Int64(time.Now().Unix()),
					}
				}
				b.StartTimer()

				for j := 0; j < b.N; j++ {
					res, err := ls.ListResources(&rdspb.ListResourcesRequest{
						Filter: filters,
					})

					if err != nil {
						b.Errorf("Unexpected error while listing resources: %v", err)
					}

					if filters == nil && len(res.GetResources()) != n {
						b.Errorf("Got %d resources, wanted: %d", len(res.GetResources()), n)
					}
				}
			})
		}
	}
}

func testModTimeCheckBehavior(t *testing.T, disableModTimeCheck bool) {
	t.Helper()
	// Set up test file.
	tf, err := ioutil.TempFile("", "cloudprober_rds_file.*.json")
	if err != nil {
		t.Fatal(err)
	}

	testFile := tf.Name()
	defer os.Remove(tf.Name())

	b, err := ioutil.ReadFile(testResourcesFiles["json"][0])
	if err != nil {
		t.Fatal(err)
	}
	ioutil.WriteFile(testFile, b, 0)

	ls, err := newLister(testFile, &configpb.ProviderConfig{
		DisableModifiedTimeCheck: proto.Bool(disableModTimeCheck),
	}, nil)
	if err != nil {
		t.Fatalf("Error creating file lister: %v", err)
	}

	// Step 1: Very first run. File should be loaded.
	res, err := ls.ListResources(nil)
	if err != nil {
		t.Errorf("Unexxpected error: %v", err)
	}
	if len(res.GetResources()) == 0 {
		t.Error("Got no resources.")
	}
	wantResources := res
	firstUpdateTime := ls.lastUpdated

	// Step 2: 2nd run. File shouldn't reload unless disableModTimeCheck is true.
	// Wait for a second and refresh again.
	time.Sleep(time.Second)
	ls.refresh()

	if !disableModTimeCheck {
		if ls.lastUpdated != firstUpdateTime {
			t.Errorf("File unexpectedly reloaded. Update time: %v, last update time: %v", ls.lastUpdated, firstUpdateTime)
		}
	} else {
		if ls.lastUpdated == firstUpdateTime {
			t.Errorf("File unexpectly didn't reload. Update time: %v, last update time: %v", ls.lastUpdated, firstUpdateTime)
		}
	}
	res, err = ls.ListResources(nil)
	if err != nil {
		t.Errorf("Unexxpected error: %v", err)
	}
	if !proto.Equal(res, wantResources) {
		t.Errorf("Got resources:\n%s\nWant resources:\n%s", res.String(), wantResources.String())
	}

	// Step 3: Third run. It should reload file.
	// Update file's modified time and see if file is reloaded.
	fileModTime := time.Now()
	if err := os.Chtimes(testFile, fileModTime, fileModTime); err != nil {
		t.Logf("Error setting modified time on the test file: %v. Finishing test early.", err)
		return
	}
	ls.refresh()

	if ls.lastUpdated.Before(fileModTime) {
		t.Errorf("File lister last update time (%v) before file mod time (%v)", ls.lastUpdated, fileModTime)
	}
	res, err = ls.ListResources(nil)
	if err != nil {
		t.Errorf("Unexxpected error: %v", err)
	}
	if !proto.Equal(res, wantResources) {
		t.Errorf("Got resources:\n%s\nWant resources:\n%s", res.String(), wantResources.String())
	}
}

func TestModTimeCheckBehavior(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		testModTimeCheckBehavior(t, false)
	})

	t.Run("ignore-mod-time", func(t *testing.T) {
		testModTimeCheckBehavior(t, true)
	})
}
