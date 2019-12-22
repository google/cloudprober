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

// This file defines functions to work with the metrics generated from the
// external probe process output.

package payload

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/metrics/payload/proto"
)

// Parser encapsulates the config for parsing payloads to metrics.
type Parser struct {
	defaultEM *metrics.EventMetrics
	aggregate bool
	l         *logger.Logger
}

// NewParser returns a new payload parser, based on the config provided.
func NewParser(opts *configpb.OutputMetricsOptions, ptype, probeName string, defaultKind metrics.Kind, l *logger.Logger) (*Parser, error) {
	parser := &Parser{
		aggregate: opts.GetAggregateInCloudprober(),
		l:         l,
	}

	em := metrics.NewEventMetrics(time.Now()).
		AddLabel("ptype", ptype).
		AddLabel("probe", probeName)

	switch opts.GetMetricsKind() {
	case configpb.OutputMetricsOptions_CUMULATIVE:
		em.Kind = metrics.CUMULATIVE
	case configpb.OutputMetricsOptions_GAUGE:
		if opts.GetAggregateInCloudprober() {
			return nil, errors.New("payload.NewParser: invalid config, GAUGE metrics should not have aggregate_in_cloudprober enabled")
		}
		em.Kind = metrics.GAUGE
	case configpb.OutputMetricsOptions_UNDEFINED:
		em.Kind = defaultKind
	}

	// Labels are specified in the probe config.
	if opts.GetAdditionalLabels() != "" {
		for _, label := range strings.Split(opts.GetAdditionalLabels(), ",") {
			labelKV := strings.Split(label, "=")
			if len(labelKV) != 2 {
				return nil, fmt.Errorf("payload.NewParser: invlaid config, wrong label format: %v", labelKV)
			}
			em.AddLabel(labelKV[0], labelKV[1])
		}
	}

	// If there are any distribution metrics, build them now itself.
	for name, distMetric := range opts.GetDistMetric() {
		d, err := metrics.NewDistributionFromProto(distMetric)
		if err != nil {
			return nil, err
		}
		em.AddMetric(name, d)
	}

	parser.defaultEM = em

	return parser, nil
}

func updateMetricValue(mv metrics.Value, val string) error {
	// If a distribution, process it through processDistValue.
	if mVal, ok := mv.(*metrics.Distribution); ok {
		if err := processDistValue(mVal, val); err != nil {
			return fmt.Errorf("error parsing distribution value (%s): %v", val, err)
		}
		return nil
	}

	v, err := metrics.ParseValueFromString(val)
	if err != nil {
		return fmt.Errorf("error parsing value (%s): %v", val, err)
	}

	return mv.Add(v)
}

// PayloadMetrics parses the given payload and either updates the provided
// metrics (if we are aggregating in cloudprober) or returns new metrics (if
// not aggregating or provided metrics are nil)
func (p *Parser) PayloadMetrics(em *metrics.EventMetrics, payload, target string) *metrics.EventMetrics {
	// If not initialized yet or not aggregating in cloudprober, initialize
	// metrics from the default metrics.
	if em == nil || !p.aggregate {
		em = p.defaultEM.Clone().AddLabel("dst", target)
	}

	em.Timestamp = time.Now()

	// Convert payload variables into metrics. Variables are specified in
	// the following format:
	// var1 value1
	// var2 value2
	for _, line := range strings.Split(payload, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		varKV := strings.Fields(line)
		if len(varKV) != 2 {
			p.l.Warningf("Wrong var key-value format: %s", line)
			continue
		}
		metricName := varKV[0]
		val := varKV[1]

		// If a metric already exists in the EventMetric (it will happen in two cases
		// -- a) we are aggregating in cloudprober, or, b) this is a distribution
		// metric that is already defined through the config), we simply add the new
		// value (after parsing) to it, except for the distributions, which are
		// handled in a special manner as their values can be provided in multiple
		// ways.
		if mv := em.Metric(metricName); mv != nil {
			if err := updateMetricValue(mv, val); err != nil {
				p.l.Warningf("Error updating metric %s with val %s: %v", metricName, val, err)
			}
			continue
		}

		// New metric name, make sure it's not disallowed.
		switch metricName {
		case "success", "total", "latency":
			p.l.Warningf("Metric name (%s) in the output conflicts with standard metrics: (success,total,latency). Ignoring.", metricName)
			continue
		}

		v, err := metrics.ParseValueFromString(val)
		if err != nil {
			p.l.Warningf("Could not parse value (%s) for the new metric name (%s): %v", val, metricName, err)
			continue
		}
		em.AddMetric(metricName, v)
	}

	return em
}

// processDistValue processes a distribution value. It works with distribution
// values in 2 formats:
// a) a full distribution string, capturing all the details, e.g.
//    "dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12"
// a) a comma-separated list of floats, where distribution details have been
//    provided at the time of config, e.g.
//    "12,13,10.1,9.875,11.1"
func processDistValue(mVal *metrics.Distribution, val string) error {
	if val[0] == 'd' {
		distVal, err := metrics.ParseDistFromString(val)
		if err != nil {
			return err
		}
		return mVal.Add(distVal)
	}

	// It's a pre-defined distribution metric
	for _, s := range strings.Split(val, ",") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("unsupported value for distribution metric (expected comma separated list of float64s): %s", val)
		}
		mVal.AddFloat64(f)
	}
	return nil
}
