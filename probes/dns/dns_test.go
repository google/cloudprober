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

package dns

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/probes/common/statskeeper"
	configpb "github.com/google/cloudprober/probes/dns/proto"
	"github.com/google/cloudprober/probes/options"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/validators"
	validatorpb "github.com/google/cloudprober/validators/proto"
	"github.com/miekg/dns"
)

// If question contains a bad domain or type, DNS query response status should
// contain an error.
const (
	questionBadDomain    = "nosuchname"
	questionBadType      = configpb.QueryType_CAA
	answerContent        = " 3600 IN A 192.168.0.1"
	answerMatchPattern   = "3600"
	answerNoMatchPattern = "NAA"
)

var (
	globalLog = logger.Logger{}
)

type mockClient struct{}

// Exchange implementation that returns an error status if the query is for
// questionBad[Domain|Type]. This allows us to check if query parameters are
// populated correctly.
func (*mockClient) Exchange(in *dns.Msg, fullTarget string) (*dns.Msg, time.Duration, error) {
	if fullTarget != "8.8.8.8:53" {
		return nil, 0, fmt.Errorf("unexpected target: %v", fullTarget)
	}
	out := &dns.Msg{}
	question := in.Question[0]
	if question.Name == questionBadDomain+"." || int(question.Qtype) == int(questionBadType) {
		out.Rcode = dns.RcodeNameError
	}
	answerStr := question.Name + answerContent
	a, err := dns.NewRR(answerStr)
	if err != nil {
		globalLog.Errorf("Error parsing answer \"%s\": %v", answerStr, err)
	} else {
		out.Answer = []dns.RR{a}
	}
	return out, time.Millisecond, nil
}
func (*mockClient) setReadTimeout(time.Duration) {}
func (*mockClient) setSourceIP(net.IP)           {}

func runProbe(t *testing.T, testName string, p *Probe, resolveF resolveFunc, total, success int64) {
	p.client = new(mockClient)
	p.targets = p.opts.Targets.ListEndpoints()

	resultsChan := make(chan statskeeper.ProbeResult, len(p.targets))
	p.runProbe(resultsChan, resolveF)

	// The resultsChan output iterates through p.targets in the same order.
	for _, target := range p.targets {
		r := <-resultsChan
		result := r.(probeRunResult)
		if result.total.Int64() != total || result.success.Int64() != success {
			t.Errorf("test(%s): result mismatch got (total, success) = (%d, %d), want (%d, %d)",
				testName, result.total.Int64(), result.success.Int64(), total, success)
		}
		if result.Target() != target.Name {
			t.Errorf("test(%s): unexpected target in probe result. got: %s, want: %s",
				testName, result.Target(), target.Name)
		}
	}
}

func TestRun(t *testing.T) {
	p := &Probe{}
	opts := &options.Options{
		Targets:   targets.StaticTargets("8.8.8.8"),
		Interval:  2 * time.Second,
		Timeout:   time.Second,
		ProbeConf: &configpb.ProbeConf{},
	}
	if err := p.Init("dns_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	runProbe(t, "basic", p, nil, 1, 1)
}

func TestResolveFirst(t *testing.T) {
	p := &Probe{}
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets("foo")
	opts.ProbeConf = &configpb.ProbeConf{ResolveFirst: proto.Bool(true)}
	if err := p.Init("dns_test_resolve_first", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		resolveF := func(target string, ipVer int) (net.IP, error) {
			if target == "foo" {
				return net.ParseIP("8.8.8.8"), nil
			}
			return nil, fmt.Errorf("resolve error")
		}
		runProbe(t, "resolve_first_success", p, resolveF, 1, 1)
	})

	t.Run("error", func(t *testing.T) {
		resolveF := func(target string, ipVer int) (net.IP, error) {
			return nil, fmt.Errorf("resolve error")
		}
		runProbe(t, "resolve_first_error", p, resolveF, 1, 0)
	})
}

func TestProbeType(t *testing.T) {
	p := &Probe{}
	badType := questionBadType
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &configpb.ProbeConf{
			QueryType: &badType,
		},
	}
	if err := p.Init("dns_probe_type_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	runProbe(t, "probetype", p, nil, 1, 0)
}

func TestBadName(t *testing.T) {
	p := &Probe{}
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &configpb.ProbeConf{
			ResolvedDomain: proto.String(questionBadDomain),
		},
	}
	if err := p.Init("dns_bad_domain_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	runProbe(t, "baddomain", p, nil, 1, 0)
}

func TestAnswerCheck(t *testing.T) {
	p := &Probe{}
	opts := &options.Options{
		Targets:  targets.StaticTargets("8.8.8.8"),
		Interval: 2 * time.Second,
		Timeout:  time.Second,
		ProbeConf: &configpb.ProbeConf{
			MinAnswers: proto.Uint32(1),
		},
	}
	if err := p.Init("dns_probe_answer_check_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	// expect success minAnswers == num answers returned == 1.
	runProbe(t, "matchminanswers", p, nil, 1, 1)

	opts.ProbeConf = &configpb.ProbeConf{
		MinAnswers: proto.Uint32(2),
	}
	if err := p.Init("dns_probe_answer_check_test", opts); err != nil {
		t.Fatalf("Error creating probe: %v", err)
	}
	// expect failure because only one answer returned and two wanted.
	runProbe(t, "toofewanswers", p, nil, 1, 0)
}

func TestValidator(t *testing.T) {
	p := &Probe{}
	for _, tst := range []struct {
		name      string
		pattern   string
		successCt int64
	}{
		{"match", answerMatchPattern, 1},
		{"nomatch", answerNoMatchPattern, 0},
	} {
		valPb := []*validatorpb.Validator{{Name: proto.String(tst.name), Type: &validatorpb.Validator_Regex{tst.pattern}}}
		validator, err := validators.Init(valPb, nil)
		if err != nil {
			t.Fatalf("Error initializing validator for pattern %v: %v", tst.pattern, err)
		}
		opts := &options.Options{
			Targets:    targets.StaticTargets("8.8.8.8"),
			Interval:   2 * time.Second,
			Timeout:    time.Second,
			ProbeConf:  &configpb.ProbeConf{},
			Validators: validator,
		}
		if err := p.Init("dns_probe_answer_"+tst.name, opts); err != nil {
			t.Fatalf("Error creating probe: %v", err)
		}
		runProbe(t, tst.name, p, nil, 1, tst.successCt)
	}
}
