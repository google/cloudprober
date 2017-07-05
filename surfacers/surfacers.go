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

// initSurfacer initializes and returns a new surfacer based on the config.
func initSurfacer(s *SurfacerDef) (Surfacer, error) {
	switch s.GetType() {
	case Type_PROMETHEUS:
		return prometheus.New(context.TODO(), s.GetName(), s.GetPrometheusSurfacer())
	case Type_STACKDRIVER:
		return stackdriver.New(s.GetName(), s.GetStackdriverSurfacer())
	case Type_FILE:
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

func defaultSurfacers() ([]Surfacer, error) {
	defaultPromSurfacer, err := prometheus.New(context.TODO(), "", nil)
	return []Surfacer{defaultPromSurfacer}, err
}

// Init initializes the surfacers from the config protobufs and returns them as
// a list.
func Init(sDefs []*SurfacerDef) ([]Surfacer, error) {
	// If no surfacers are defined, return default surfacers. This behavior
	// can be disabled by explicitly specifying "surfacer {}" in the config.
	if len(sDefs) == 0 {
		return defaultSurfacers()
	}

	var result []Surfacer
	for _, sDef := range sDefs {
		if sDef.GetType() == Type_NONE {
			continue
		}
		s, err := initSurfacer(sDef)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}
