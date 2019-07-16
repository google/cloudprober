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

package external

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/probes/external/proto"
)

func (p *Probe) initPayloadMetrics() error {
	if !p.c.GetOutputAsMetrics() {
		return nil
	}

	opts := p.c.GetOutputMetricsOptions()

	em := metrics.NewEventMetrics(time.Now()).
		AddLabel("ptype", "external").
		AddLabel("probe", p.name)

	switch opts.GetMetricsKind() {
	case configpb.OutputMetricsOptions_CUMULATIVE:
		em.Kind = metrics.CUMULATIVE
	case configpb.OutputMetricsOptions_GAUGE:
		if opts.GetAggregateInCloudprober() {
			return errors.New("invalid config: GAUGE metrics should not have aggregate_in_cloudprober enabled")
		}
		em.Kind = metrics.GAUGE
	case configpb.OutputMetricsOptions_UNDEFINED:
		if p.c.GetMode() == configpb.ProbeConf_ONCE {
			em.Kind = metrics.GAUGE
		} else {
			em.Kind = metrics.CUMULATIVE
		}
	}

	// Labels are specified in the probe config.
	if opts.GetAdditionalLabels() != "" {
		for _, label := range strings.Split(opts.GetAdditionalLabels(), ",") {
			labelKV := strings.Split(label, "=")
			if len(labelKV) != 2 {
				p.l.Warningf("Wrong label format: %s", labelKV)
				continue
			}
			em.AddLabel(labelKV[0], labelKV[1])
		}
	}

	// If there are any distribution metrics, build them now itself.
	for name, distMetric := range opts.GetDistMetric() {
		d, err := metrics.NewDistributionFromProto(distMetric)
		if err != nil {
			return err
		}
		em.AddMetric(name, d)
	}
	p.defaultPayloadMetrics = em
	return nil
}

func (p *Probe) payloadToMetrics(target, payload string, result *result) *metrics.EventMetrics {
	em := result.payloadMetrics
	if em == nil {
		// result will have a nil payloadMetrics only if we are not aggregating in
		// cloudprober.
		em = p.defaultPayloadMetrics.Clone().AddLabel("dst", target)
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
			// If a distribution, process it through processDistValue.
			if mVal, ok := mv.(*metrics.Distribution); ok {
				if err := processDistValue(mVal, val); err != nil {
					p.l.Warningf("Error parsing distribution value (%s) for metric (%s) in probe payload: %v", val, metricName, err)
					continue
				}
			}

			v, err := parseValue(val, p.c.GetOutputMetricsOptions().GetAggregateInCloudprober())
			if err != nil {
				p.l.Warningf("Error parsing value (%s) for metric (%s) in probe payload: %v", val, metricName, err)
				continue
			}
			mv.Add(v)
		}

		// New metric name, make sure it's not disallowed.
		switch metricName {
		case "success", "total", "latency":
			p.l.Warningf("Metric name (%s) in the output conflicts with standard metrics: (success,total,latency). Ignoring.", metricName)
			continue
		}

		v, err := parseValue(val, p.c.GetOutputMetricsOptions().GetAggregateInCloudprober())
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

func parseValue(val string, aggregationEnabled bool) (metrics.Value, error) {
	c := val[0]
	switch {
	// A float value
	case '0' <= c && c <= '9':
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, err
		}
		return metrics.NewFloat(f), nil

	// A map value
	case c == 'm':
		if !strings.HasPrefix(val, "map") {
			break
		}
		return metrics.ParseMapFromString(val)

	// A string value
	case c == '"':
		if aggregationEnabled {
			return nil, fmt.Errorf("string metric value (%s) is incompatible with aggregate_in_cloudprober option", val)
		}
		return metrics.NewString(strings.Trim(val, "\"")), nil

	// A distribution value
	case c == 'd':
		if !strings.HasPrefix(val, "dist") {
			break
		}
		distVal, err := metrics.ParseDistFromString(val)
		if err != nil {
			return nil, err
		}
		return distVal, nil
	}

	return nil, errors.New("unknown value type")
}
