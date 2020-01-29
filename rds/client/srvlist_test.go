// Copyright 2020 Google Inc.
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

package client

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

var testDefaultPort = "9999"

func TestParseAddr(t *testing.T) {
	var tests = []struct {
		addr, host, port string
		err              error
	}{
		{
			addr: "rds-service:443",
			host: "rds-service",
			port: "443",
			err:  nil,
		},
		{
			addr: "192.168.1.2:4430",
			host: "192.168.1.2",
			port: "4430",
			err:  nil,
		},
		{
			addr: "192.168.1.4",
			host: "192.168.1.4",
			port: testDefaultPort,
			err:  nil,
		},
		{
			addr: "1620:15c:2c4:201::ff",
			host: "[1620:15c:2c4:201::ff]",
			port: testDefaultPort,
			err:  nil,
		},
		{
			addr: "rds-service:",
			host: "rds-service",
			port: testDefaultPort,
			err:  nil,
		},
		{
			addr: ":9314",
			host: "localhost",
			port: "9314",
			err:  nil,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("parsing %s", test.addr), func(t *testing.T) {
			host, port, err := parseAddr(test.addr, testDefaultPort)

			if host != test.host || port != test.port || err != test.err {
				t.Errorf("parseAddr(%s)=%s, %s, %v, want=%s, %s, %v", test.addr, host, port, err, test.host, test.port, test.err)
			}
		})
	}
}

func TestNewResolver(t *testing.T) {
	var tests = []struct {
		target string
		hosts  []string
		ports  []string
	}{
		{
			target: "rds-service-a:443,rds-service-b:9314",
			hosts:  []string{"rds-service-a", "rds-service-b"},
			ports:  []string{"443", "9314"},
		},
		{
			target: "35.14.14.1:443,rds-service-b:9314",
			hosts:  []string{"35.14.14.1", "rds-service-b"},
			ports:  []string{"443", "9314"},
		},
		{
			target: "35.14.14.1,35.14.14.2",
			hosts:  []string{"35.14.14.1", "35.14.14.2"},
			ports:  []string{testDefaultPort, testDefaultPort},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("parsing %s", test.target), func(t *testing.T) {
			res, err := newSrvListResolver(test.target, testDefaultPort)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			sort.Strings(res.hostList)
			sort.Strings(test.hosts)
			if !reflect.DeepEqual(res.hostList, test.hosts) {
				t.Errorf("res.hostList, got=%v, want=%v", res.hostList, test.hosts)
			}

			sort.Strings(res.portList)
			sort.Strings(test.ports)
			if !reflect.DeepEqual(res.portList, test.ports) {
				t.Errorf("res.portList, got=%v, want=%v", res.portList, test.ports)
			}
		})
	}

}
