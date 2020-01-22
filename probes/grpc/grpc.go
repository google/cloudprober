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

/*
Package grpc implements a gRPC probe.

This probes a cloudprober gRPC server and reports success rate, latency, and
connection failures.
*/
package grpc

import (
	"context"
	"errors"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/grpc/proto"
	"github.com/google/cloudprober/probes/options"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	return errors.New("gRPC probe not yet implemented")
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	// TODO(ls692): Add probe implementation.
	return
}
