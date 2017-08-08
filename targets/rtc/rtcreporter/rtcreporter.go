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

// Package rtcreporter implements a reporting mechanism for RTC targets.
// The RtcReportOptions configures how a cloudprober instance should populate
// an RtcTargetInfo protobuf to report to a set of RTC configurations. For more
// information, see cloudprober/targets/rtc to see how target listing works for
// RTC configs.
//
// When reporting attributes such as public/private ip, the sysVars map will be
// used. For more info see cloudprober/util.
package rtcreporter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/targets/rtc/rtcservice"
)

// Reporter provides the means and configuration for a cloudprober instance to
// report its sysVars to a set of RTC configs.
type Reporter struct {
	pb         *RtcReportOptions
	cfgs       []rtcservice.Config
	sysVars    map[string]string
	reportVars []string
	groups     []string
	l          *logger.Logger
}

// New returns a Reporter from a provided configuration. An error will be returned
// if any of the report_variables are not defined in sysVars.
func New(pb *RtcReportOptions, sysVars map[string]string, l *logger.Logger) (*Reporter, error) {
	proj, ok := sysVars["project"]
	if !ok {
		return nil, errors.New("rtcserve.New: sysVars has no \"project\"")
	}

	var cfgs []rtcservice.Config
	for _, name := range pb.GetCfgs() {
		c, err := rtcservice.New(proj, name)
		if err != nil {
			return nil, err
		}
		cfgs = append(cfgs, c)
	}
	reportVars := pb.GetVariables()
	groups := pb.GetGroups()

	// Check if the report vars exist, or fail fast
	for _, v := range reportVars {
		if _, ok := sysVars[v]; !ok {
			return nil, fmt.Errorf("rtcserve.New() : No sysVar %v", v)
		}
	}

	return &Reporter{
		pb:         pb,
		cfgs:       cfgs,
		sysVars:    sysVars,
		reportVars: reportVars,
		groups:     groups,
		l:          l,
	}, nil
}

// Start calls report for every RTC config each tick of the clock.
func (r *Reporter) Start(ctx context.Context) {
	t := time.NewTicker(time.Millisecond * time.Duration(r.pb.GetIntervalMsec()))
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, c := range r.cfgs {
				if err := r.report(c); err != nil {
					r.l.Errorf("Unable to report to RTC config: %v", err)
				}
			}
		}
	}
}

func (r *Reporter) mkTargetInfo() *RtcTargetInfo {
	pb := &RtcTargetInfo{
		InstanceName: proto.String(r.sysVars["hostname"]),
		Groups:       r.groups,
	}

	addresses := make([]*RtcTargetInfo_Address, len(r.reportVars))
	for i, v := range r.reportVars {
		addresses[i] = &RtcTargetInfo_Address{
			Tag:     proto.String(v),
			Address: proto.String(r.sysVars[v]),
		}
	}
	pb.Addresses = addresses

	return pb
}

func (r *Reporter) report(c rtcservice.Config) error {
	pb := r.mkTargetInfo()
	data, err := proto.Marshal(pb)
	if err != nil {
		return fmt.Errorf("rtcserve.report(): unable to marshal RtcTargetInfo: %v", err)
	}
	return c.Write(pb.GetInstanceName(), data)
}
