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

/*
Package options provides a shared interface to common probe options.
*/
package options

import (
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/targets"
	"github.com/google/cloudprober/validators"
)

// Options encapsulates common probe options.
type Options struct {
	Targets           targets.Targets
	Interval, Timeout time.Duration
	Logger            *logger.Logger
	ProbeConf         interface{} // Probe-type specific config
	LatencyDist       *metrics.Distribution
	LatencyUnit       time.Duration
	Validators        []*validators.ValidatorWithName
}
