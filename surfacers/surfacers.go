// Copyright 2017-2021 The Cloudprober Authors.
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
	"html/template"
	"strings"
	"sync"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/surfacers/cloudwatch"
	"github.com/google/cloudprober/surfacers/common/options"
	"github.com/google/cloudprober/surfacers/file"
	"github.com/google/cloudprober/surfacers/postgres"
	"github.com/google/cloudprober/surfacers/prometheus"
	"github.com/google/cloudprober/surfacers/pubsub"
	"github.com/google/cloudprober/surfacers/stackdriver"
	"github.com/google/cloudprober/surfacers/datadog"
	"github.com/google/cloudprober/web/formatutils"

	surfacerpb "github.com/google/cloudprober/surfacers/proto"
	surfacerspb "github.com/google/cloudprober/surfacers/proto"
)

var (
	userDefinedSurfacers   = make(map[string]Surfacer)
	userDefinedSurfacersMu sync.Mutex
)

// StatusTmpl variable stores the HTML template suitable to generate the
// surfacers' status for cloudprober's /status page. It expects an array of
// SurfacerInfo objects as input.
var StatusTmpl = template.Must(template.New("statusTmpl").Parse(`
<table class="status-list">
  <tr>
    <th>Type</th>
    <th>Name</th>
    <th>Conf</th>
  </tr>
  {{ range . }}
  <tr>
    <td>{{.Type}}</td>
    <td>{{.Name}}</td>
    <td>
    {{if .Conf}}
      <pre>{{.Conf}}</pre>
    {{else}}
      default
    {{end}}
    </td>
  </tr>
  {{ end }}
</table>
`))

// Default surfacers. These surfacers are enabled if no surfacer is defined.
var defaultSurfacers = []*surfacerpb.SurfacerDef{
	&surfacerpb.SurfacerDef{
		Type: surfacerpb.Type_PROMETHEUS.Enum(),
	},
	&surfacerpb.SurfacerDef{
		Type: surfacerpb.Type_FILE.Enum(),
	},
}

// Surfacer is an interface for all metrics surfacing systems
type Surfacer interface {
	// Function for writing a piece of metric data to a specified metric
	// store (or other location).
	Write(ctx context.Context, em *metrics.EventMetrics)
}

type surfacerWrapper struct {
	Surfacer
	opts *options.Options
}

func (sw *surfacerWrapper) Write(ctx context.Context, em *metrics.EventMetrics) {
	if sw.opts.AllowEventMetrics(em) {
		sw.Surfacer.Write(ctx, em)
	}
}

// SurfacerInfo encapsulates a Surfacer and related info.
type SurfacerInfo struct {
	Surfacer
	Type string
	Name string
	Conf string
}

func inferType(s *surfacerpb.SurfacerDef) surfacerspb.Type {
	switch s.Surfacer.(type) {
	case *surfacerpb.SurfacerDef_PrometheusSurfacer:
		return surfacerspb.Type_PROMETHEUS
	case *surfacerpb.SurfacerDef_StackdriverSurfacer:
		return surfacerspb.Type_STACKDRIVER
	case *surfacerpb.SurfacerDef_FileSurfacer:
		return surfacerspb.Type_FILE
	case *surfacerpb.SurfacerDef_PostgresSurfacer:
		return surfacerspb.Type_POSTGRES
	case *surfacerpb.SurfacerDef_PubsubSurfacer:
		return surfacerspb.Type_PUBSUB
	case *surfacerpb.SurfacerDef_CloudwatchSurfacer:
		return surfacerspb.Type_CLOUDWATCH
	case *surfacerpb.SurfacerDef_DatadogSurfacer:
		return surfacerspb.Type_DATADOG
	}

	return surfacerspb.Type_NONE
}

// initSurfacer initializes and returns a new surfacer based on the config.
func initSurfacer(ctx context.Context, s *surfacerpb.SurfacerDef, sType surfacerspb.Type) (Surfacer, interface{}, error) {
	// Create a new logger
	logName := s.GetName()
	if logName == "" {
		logName = strings.ToLower(s.GetType().String())
	}

	l, err := logger.NewCloudproberLog(logName)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create cloud logger: %v", err)
	}

	opts, err := options.BuildOptionsFromConfig(s)
	if err != nil {
		return nil, nil, err
	}

	var conf interface{}
	var surfacer Surfacer

	switch sType {
	case surfacerpb.Type_PROMETHEUS:
		surfacer, err = prometheus.New(ctx, s.GetPrometheusSurfacer(), opts, l)
		conf = s.GetPrometheusSurfacer()
	case surfacerpb.Type_STACKDRIVER:
		surfacer, err = stackdriver.New(ctx, s.GetStackdriverSurfacer(), opts, l)
		conf = s.GetStackdriverSurfacer()
	case surfacerpb.Type_FILE:
		surfacer, err = file.New(ctx, s.GetFileSurfacer(), opts, l)
		conf = s.GetFileSurfacer()
	case surfacerpb.Type_POSTGRES:
		surfacer, err = postgres.New(ctx, s.GetPostgresSurfacer(), l)
		conf = s.GetPostgresSurfacer()
	case surfacerpb.Type_PUBSUB:
		surfacer, err = pubsub.New(ctx, s.GetPubsubSurfacer(), opts, l)
		conf = s.GetPubsubSurfacer()
	case surfacerpb.Type_CLOUDWATCH:
		surfacer, err = cloudwatch.New(ctx, s.GetCloudwatchSurfacer(), opts, l)
		conf = s.GetCloudwatchSurfacer()
	case surfacerpb.Type_DATADOG:
		surfacer, err = datadog.New(ctx, s.GetDatadogSurfacer(), opts, l)
		conf = s.GetDatadogSurfacer()
	case surfacerpb.Type_USER_DEFINED:
		userDefinedSurfacersMu.Lock()
		defer userDefinedSurfacersMu.Unlock()
		surfacer = userDefinedSurfacers[s.GetName()]
		if surfacer == nil {
			return nil, nil, fmt.Errorf("unregistered user defined surfacer: %s", s.GetName())
		}
	default:
		return nil, nil, fmt.Errorf("unknown surfacer type: %s", s.GetType())
	}

	return &surfacerWrapper{
		Surfacer: surfacer,
		opts:     opts,
	}, conf, err
}

// Init initializes the surfacers from the config protobufs and returns them as
// a list.
func Init(ctx context.Context, sDefs []*surfacerpb.SurfacerDef) ([]*SurfacerInfo, error) {
	// If no surfacers are defined, return default surfacers. This behavior
	// can be disabled by explicitly specifying "surfacer {}" in the config.
	if len(sDefs) == 0 {
		sDefs = defaultSurfacers
	}

	var result []*SurfacerInfo
	for _, sDef := range sDefs {
		sType := sDef.GetType()

		if sType == surfacerpb.Type_NONE {
			// Don't do anything if surfacer type is NONE and nothing is defined inside
			// it: for example: "surfacer{}". This is one of the ways to disable
			// surfacers as not adding surfacers at all results in default surfacers
			// being added automatically.
			if sDef.Surfacer == nil {
				continue
			}
			sType = inferType(sDef)
		}

		s, conf, err := initSurfacer(ctx, sDef, sType)
		if err != nil {
			return nil, err
		}

		result = append(result, &SurfacerInfo{
			Surfacer: s,
			Type:     sType.String(),
			Name:     sDef.GetName(),
			Conf:     formatutils.ConfToString(conf),
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
