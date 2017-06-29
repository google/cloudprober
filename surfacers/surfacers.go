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

/*
Package surfacers is the base package for creating Surfacer objects that are
used for writing metics data to different monitoring services.

Any Surfacer that is created for writing metrics data to a monitor system
should implement the below Surfacer interface and should accept
metrics.EventMetrics object through a Write() call. Each new surfacer should
also plug itself in through the New() method defined here.
*/
package surfacers

import (
	"context"
	"fmt"

	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/surfacers/file"
	"github.com/google/cloudprober/surfacers/prometheus"
	"github.com/google/cloudprober/surfacers/stackdriver"
)

// New returns a new surfacer based on the config
func New(s *SurfacerDef) (Surfacer, error) {
	switch s.GetType() {
	case SurfacerDef_PROMETHEUS:
		return prometheus.New(context.TODO(), s.GetName(), s.GetPrometheusSurfacer())
	case SurfacerDef_STACKDRIVER:
		return stackdriver.New(s.GetName(), s.GetStackdriverSurfacer())
	case SurfacerDef_FILE:
		return file.New(s.GetName(), s.GetFileSurfacer())
	default:
		return nil, fmt.Errorf("unknown surfacer type: %s", s.GetType())
	}
}

// Surfacer is the base class for all metrics surfacing systems
type Surfacer interface {
	// Function for writing a piece of metric data to a specified metric
	// store (or other location).
	Write(ctx context.Context, em *metrics.EventMetrics)
}
