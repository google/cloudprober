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

package dns

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/targets"
	"github.com/miekg/dns"
)

type mockClient struct{}

func (*mockClient) Exchange(*dns.Msg, string) (*dns.Msg, time.Duration, error) {
	return new(dns.Msg), time.Millisecond, nil
}
func (*mockClient) SetReadTimeout(time.Duration) {}

func TestRun(t *testing.T) {
	c := &ProbeConf{
		StatsExportIntervalMsec: proto.Int32(1000),
	}

	p := &Probe{}

	tgts := targets.StaticTargets("8.8.8.8")
	p.Init("dns_test", tgts, 2*time.Second, time.Second, nil, c)
	p.client = new(mockClient)
	p.targets = p.tgts.List()

	resultsChan := make(chan probeutils.ProbeResult, len(p.targets))
	p.runProbe(resultsChan)

	// Strings that should be in all targets' output.
	reqStrs := map[string]int64{
		"sent": 1,
		"rcvd": 1,
	}

	// The resultsChan output iterates through p.targets in the same order.
	for _, target := range p.targets {
		r := <-resultsChan
		result := r.(probeRunResult)
		if result.sent.Int64() != reqStrs["sent"] || result.rcvd.Int64() != reqStrs["rcvd"] {
			t.Errorf("Mismatch got (sent, rcvd) = (%d, %d), want (%d, %d)", result.sent.Int64(), result.rcvd.Int64(), reqStrs["sent"], reqStrs["rcvd"])
		}
		if result.Target() != target {
			t.Errorf("Unexpected target in probe result. Got: %s, Expected: %s", result.Target(), target)
		}
	}
	p.runProbe(resultsChan)
}
