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
	configpb "github.com/google/cloudprober/probes/dns/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/targets"
	"github.com/miekg/dns"
)

// If question contains a bad domain or type, DNS query response status should
// contain an error.
const (
	questionBadDomain = "nosuchname"
	questionBadType   = configpb.QueryType_CAA
)

type mockClient struct{}

// Exchange implementation that returns an error status if the query is for
// questionBad[Domain|Type]. This allows us to check if query parameters are
// populated correctly.
func (*mockClient) Exchange(in *dns.Msg, _ string) (*dns.Msg, time.Duration, error) {
	out := &dns.Msg{}
	question := in.Question[0]
	if question.Name == questionBadDomain+"." || int(question.Qtype) == int(questionBadType) {
		out.Rcode = dns.RcodeNameError
	}
	return out, time.Millisecond, nil
}
func (*mockClient) SetReadTimeout(time.Duration) {}

func runProbe(t *testing.T, p *Probe, total, success int64) {
	p.client = new(mockClient)
	p.targets = p.opts.Targets.List()

	resultsChan := make(chan probeutils.ProbeResult, len(p.targets))
	p.runProbe(resultsChan)

	// The resultsChan output iterates through p.targets in the same order.
	for _, target := range p.targets {
		r := <-resultsChan
		result := r.(probeRunResult)
		if result.total.Int64() != total || result.success.Int64() != success {
			t.Errorf("Mismatch got (total, success) = (%d, %d), want (%d, %d)", result.total.Int64(), result.success.Int64(), total, success)
		}
		if result.Target() != target {
			t.Errorf("Unexpected target in probe result. Got: %s, Expected: %s", result.Target(), target)
		}
	}
}

func TestRun(t *testing.T) {
	p := &Probe{}
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &configpb.ProbeConf{
			StatsExportIntervalMsec: proto.Int32(1000),
		},
	}
	if err := p.Init("dns_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	runProbe(t, p, 1, 1)
}

func TestProbeType(t *testing.T) {
	p := &Probe{}
	badType := questionBadType
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &configpb.ProbeConf{
			StatsExportIntervalMsec: proto.Int32(1000),
			QueryType:               &badType,
		},
	}
	if err := p.Init("dns_probe_type_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	runProbe(t, p, 1, 0)
}

func TestBadName(t *testing.T) {
	p := &Probe{}
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &configpb.ProbeConf{
			StatsExportIntervalMsec: proto.Int32(1000),
			ResolvedDomain:          proto.String(questionBadDomain),
		},
	}
	if err := p.Init("dns_bad_domain_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	runProbe(t, p, 1, 0)
}
