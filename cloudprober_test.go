// Copyright 2019-2020 The Cloudprober Authors.
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

package cloudprober

import (
	"os"
	"testing"

	configpb "github.com/google/cloudprober/config/proto"
	"google.golang.org/protobuf/proto"
)

func TestGetDefaultServerPort(t *testing.T) {
	tests := []struct {
		desc       string
		configPort int32
		envVar     string
		wantPort   int
		wantErr    bool
	}{
		{
			desc:       "use port from config",
			configPort: 9316,
			envVar:     "3141",
			wantPort:   9316,
		},
		{
			desc:       "use default port",
			configPort: 0,
			envVar:     "",
			wantPort:   DefaultServerPort,
		},
		{
			desc:       "use port from env",
			configPort: 0,
			envVar:     "3141",
			wantPort:   3141,
		},
		{
			desc:       "ignore kubernetes port",
			configPort: 0,
			envVar:     "tcp://100.101.102.103:3141",
			wantPort:   9313,
		},
		{
			desc:       "error due to bad env var",
			configPort: 0,
			envVar:     "a3141",
			wantErr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			os.Setenv(ServerPortEnvVar, test.envVar)
			port, err := getDefaultServerPort(&configpb.ProberConfig{
				Port: proto.Int32(test.configPort),
			}, nil)

			if err != nil {
				if !test.wantErr {
					t.Errorf("Got unexpected error: %v", err)
				} else {
					return
				}
			}

			if port != test.wantPort {
				t.Errorf("got port: %d, want port: %d", port, test.wantPort)
			}
		})
	}

}
