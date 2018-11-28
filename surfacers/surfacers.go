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
	"strings"
	"sync"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/surfacers/file"
	"github.com/google/cloudprober/surfacers/postgres"
	"github.com/google/cloudprober/surfacers/prometheus"
	"github.com/google/cloudprober/surfacers/stackdriver"

	surfacerpb "github.com/google/cloudprober/surfacers/proto"
)

var (
	userDefinedSurfacers   = make(map[string]Surfacer)
	userDefinedSurfacersMu sync.Mutex
)

// Default surfacers. These surfacers are enabled if no surfacer is defined.
var defaultSurfacers = []*surfacerpb.SurfacerDef{
	&surfacerpb.SurfacerDef{
		Type: surfacerpb.Type_PROMETHEUS.Enum(),
	},
	&surfacerpb.SurfacerDef{
		Type: surfacerpb.Type_FILE.Enum(),
	},
}

// initSurfacer initializes and returns a new surfacer based on the config.
func initSurfacer(s *surfacerpb.SurfacerDef) (Surfacer, interface{}, error) {
	// Create a new logger
	logName := s.GetName()
	if logName == "" {
		logName = strings.ToLower(s.GetType().String())
	}

	// TODO(manugarg): Plumb context here too.
	l, err := logger.New(context.TODO(), logName)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create cloud logger: %v", err)
	}

	var conf interface{}
	var surfacer Surfacer

	switch s.GetType() {
	case surfacerpb.Type_PROMETHEUS:
		surfacer, err = prometheus.New(s.GetPrometheusSurfacer(), l)
		conf = s.GetPrometheusSurfacer()
	case surfacerpb.Type_STACKDRIVER:
		surfacer, err = stackdriver.New(s.GetStackdriverSurfacer(), l)
		conf = s.GetStackdriverSurfacer()
	case surfacerpb.Type_FILE:
		surfacer, err = file.New(s.GetFileSurfacer(), l)
		conf = s.GetFileSurfacer()
	case surfacerpb.Type_POSTGRES:
		surfacer, err = postgres.New(s.GetPostgresSurfacer(), l)
		conf = s.GetPostgresSurfacer()
	case surfacerpb.Type_USER_DEFINED:
		userDefinedSurfacersMu.Lock()
		defer userDefinedSurfacersMu.Unlock()
		surfacer := userDefinedSurfacers[s.GetName()]
		if surfacer == nil {
			return nil, nil, fmt.Errorf("unregistered user defined surfacer: %s", s.GetName())
		}
	default:
		return nil, nil, fmt.Errorf("unknown surfacer type: %s", s.GetType())
	}

	return surfacer, conf, err
}

// Surfacer is an interface for all metrics surfacing systems
type Surfacer interface {
	// Function for writing a piece of metric data to a specified metric
	// store (or other location).
	Write(ctx context.Context, em *metrics.EventMetrics)
}

// SurfacerInfo encapsulates a Surfacer and related info.
type SurfacerInfo struct {
	Surfacer
	Type string
	Name string
	Conf string
}

// Init initializes the surfacers from the config protobufs and returns them as
// a list.
func Init(sDefs []*surfacerpb.SurfacerDef) ([]*SurfacerInfo, error) {
	// If no surfacers are defined, return default surfacers. This behavior
	// can be disabled by explicitly specifying "surfacer {}" in the config.
	if len(sDefs) == 0 {
		sDefs = defaultSurfacers
	}

	var result []*SurfacerInfo
	for _, sDef := range sDefs {
		if sDef.GetType() == surfacerpb.Type_NONE {
			continue
		}
		s, conf, err := initSurfacer(sDef)
		if err != nil {
			return nil, err
		}

		confStr := ""
		if conf != nil {
			if stringer, ok := conf.(fmt.Stringer); ok {
				confStr = stringer.String()
			}
		}

		result = append(result, &SurfacerInfo{
			Surfacer: s,
			Type:     sDef.GetType().String(),
			Name:     sDef.GetName(),
			Conf:     confStr,
		})
	}
	return result, nil
}

// Register allows you to register a user defined surfacer with cloudprober.
// Example usage:
//	import (
//		"github.com/google/cloudprober"
//		"github.com/google/cloudprober/surfacers"
//	)
//
//	s := &FancySurfacer{}
//	surfacers.Register("fancy_surfacer", s)
//	pr, err := cloudprober.InitFromConfig(*configFile)
//	if err != nil {
//		log.Exitf("Error initializing cloudprober. Err: %v", err)
//	}
func Register(name string, s Surfacer) {
	userDefinedSurfacersMu.Lock()
	defer userDefinedSurfacersMu.Unlock()
	userDefinedSurfacers[name] = s
}
