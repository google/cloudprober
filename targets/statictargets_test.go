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

package targets

import (
	"reflect"
	"testing"

	"github.com/google/cloudprober/targets/endpoint"
)

func TestStaticTargets(t *testing.T) {
	for _, test := range []struct {
		desc      string
		hosts     string
		wantNames []string
		wantPorts []int
		wantErr   bool
	}{
		{
			desc:      "valid hosts (extra space)",
			hosts:     "www.google.com ,127.0.0.1, 2001::2001",
			wantNames: []string{"www.google.com", "127.0.0.1", "2001::2001"},
			wantPorts: []int{0, 0, 0},
		},
		{
			desc:      "Ports in name",
			hosts:     "www.google.com:80,127.0.0.1:8080,[2001::2001]:8081",
			wantNames: []string{"www.google.com", "127.0.0.1", "2001::2001"},
			wantPorts: []int{80, 8080, 8081},
		},
		{
			desc:    "invalid host, IPv6 port in name without brackets",
			hosts:   "www.google.com,127.0.0.1:8080,0:0:0:0:0:1:8081",
			wantErr: true,
		},
		{
			desc:    "invalid hosts,URL in name",
			hosts:   "www.google.com/url1,127.0.0.1,2001::2001",
			wantErr: true,
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			tgts, err := staticTargets(test.hosts)
			if err != nil {
				if !test.wantErr {
					t.Errorf("Unexpected error: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Errorf("wanted error, but didn't get one")
			}
			wantEp := make([]endpoint.Endpoint, len(test.wantNames))
			for i := 0; i < len(wantEp); i++ {
				wantEp[i] = endpoint.Endpoint{Name: test.wantNames[i], Port: test.wantPorts[i]}
			}
			got := tgts.ListEndpoints()
			if !reflect.DeepEqual(got, wantEp) {
				t.Errorf("staticTargets: got=%v, wanted: %v", got, wantEp)
			}
		})
	}
}
