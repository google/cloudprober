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
	"github.com/google/cloudprober/probes/options"
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
	p := &Probe{}
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &ProbeConf{
			StatsExportIntervalMsec: proto.Int32(1000),
		},
	}
	p.Init("dns_test", opts)
	p.client = new(mockClient)
	p.targets = p.opts.Targets.List()

	resultsChan := make(chan probeutils.ProbeResult, len(p.targets))
	p.runProbe(resultsChan)

	// Strings that should be in all targets' output.
	reqStrs := map[string]int64{
		"total":   1,
		"success": 1,
	}

	// The resultsChan output iterates through p.targets in the same order.
	for _, target := range p.targets {
		r := <-resultsChan
		result := r.(probeRunResult)
		if result.total.Int64() != reqStrs["total"] || result.success.Int64() != reqStrs["success"] {
			t.Errorf("Mismatch got (total, success) = (%d, %d), want (%d, %d)", result.total.Int64(), result.success.Int64(), reqStrs["total"], reqStrs["success"])
		}
		if result.Target() != target {
			t.Errorf("Unexpected target in probe result. Got: %s, Expected: %s", result.Target(), target)
		}
	}
	p.runProbe(resultsChan)
}
