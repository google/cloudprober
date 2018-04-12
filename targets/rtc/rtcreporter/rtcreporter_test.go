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

package rtcreporter

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	rpb "github.com/google/cloudprober/targets/rtc/rtcreporter/proto"
	"github.com/google/cloudprober/targets/rtc/rtcservice"
	"github.com/kylelemons/godebug/pretty"
)

// TestMkTargetInfo tests the basic functionality of mkTargetInfo for regression
// purposes.
func TestMkTargetInfo(t *testing.T) {
	r := &Reporter{
		sysVars: map[string]string{
			"hostname":  "host1",
			"public_ip": "1.1.1.1",
		},
		reportVars: []string{"public_ip"},
		groups:     []string{"Group 1"},
	}
	pb := r.mkTargetInfo()
	if pb.GetInstanceName() != "host1" {
		t.Errorf("pb.GetInstanceName() = %q, want \"host1\".", pb.GetInstanceName())
	}
	if diff := pretty.Compare([]string{"Group 1"}, pb.GetGroups()); diff != "" {
		t.Errorf("pb.GetGroups() got diff:\n%s", diff)
	}
	addrs := pb.GetAddresses()
	if len(addrs) != 1 {
		t.Fatalf("len(addrs) = %v, want 1.", len(addrs))
	}
	if addrs[0].GetTag() != "public_ip" {
		t.Errorf("a.GetTag() = %q, want \"public_ip\".", addrs[0].GetTag())
	}
	if addrs[0].GetAddress() != "1.1.1.1" {
		t.Errorf("a.GetAddress() = %q, want \"1.1.1.1\".", addrs[0].GetAddress())
	}
}

// TestReport only tests that report puts the correct protobuf into the RTC config.
func TestReport(t *testing.T) {
	l, err := logger.New(context.Background(), "rtcserve_test")
	if err != nil {
		t.Errorf("Unable to initialize logger. %v", err)
	}
	r := &Reporter{
		sysVars: map[string]string{
			"hostname":  "host1",
			"public_ip": "1.1.1.1",
		},
		cfgs:       []rtcservice.Config{rtcservice.NewStub()},
		reportVars: []string{"public_ip"},
		groups:     []string{"Group 1"},
		l:          l,
	}

	if err := r.report(r.cfgs[0]); err != nil {
		t.Errorf("report(c) = %v, want nil.", err)
	}
	vars, err := r.cfgs[0].List()
	if err != nil {
		t.Fatalf("r.cfgs[0].List() returned error: Error %v", err)
	}
	if len(vars) != 1 {
		t.Fatalf("len(rtc.List()) = %v, want %v.", len(vars), 1)
	}
	v, err := r.cfgs[0].Val(vars[0])
	if err != nil {
		t.Fatalf("r.cfgs[0].Val(vars[0]) returned error: Error %v", err)
	}
	pb := &rpb.RtcTargetInfo{}
	if err := proto.Unmarshal(v, pb); err != nil {
		t.Fatalf("Got error unmarshaling protobuf: %v", err)
	}
	if pb.GetInstanceName() != "host1" {
		t.Errorf("pb.GetInstanceName() = %q, want \"host1\".", pb.GetInstanceName())
	}
	if diff := pretty.Compare([]string{"Group 1"}, pb.GetGroups()); diff != "" {
		t.Errorf("pb.GetGroups() got diff:\n%s", diff)
	}
	addrs := pb.GetAddresses()
	if len(addrs) != 1 {
		t.Fatalf("len(addrs) = %v, want 1.", len(addrs))
	}
	if addrs[0].GetTag() != "public_ip" {
		t.Errorf("a.GetTag() = %q, want \"public_ip\".", addrs[0].GetTag())
	}
	if addrs[0].GetAddress() != "1.1.1.1" {
		t.Errorf("a.GetAddress() = %q, want \"1.1.1.1\".", addrs[0].GetAddress())
	}
}
