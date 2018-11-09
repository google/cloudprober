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

func (p *Probe) payloadToMetrics(target, payload string) *metrics.EventMetrics {
	var em *metrics.EventMetrics
	if p.c.GetOutputMetricsOptions().GetAggregateInCloudprober() {
		// If we are aggregating in Cloudprober, we maintain an EventMetrics
		// struct per-target.
		p.payloadMetricsMu.Lock()
		em = p.payloadMetrics[target]
		if em == nil {
			em = p.defaultPayloadMetrics.Clone().AddLabel("dst", target)
			p.payloadMetrics[target] = em
		}
		p.payloadMetricsMu.Unlock()
	} else {
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

		if mv := em.Metric(metricName); mv != nil {
			switch mv.(type) {
			case *metrics.Distribution:
				for _, s := range strings.Split(val, ",") {
					f, err := strconv.ParseFloat(s, 64)
					if err != nil {
						p.l.Warningf("Unsupported value for metric %s in probe payload (expected comma separated list of float64s): %s", metricName, val)
						continue
					}
					mv.AddFloat64(f)
				}
			case *metrics.Float:
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					p.l.Warningf("Unsupported value for metric %s in probe payload (expected float64): %s", metricName, val)
					continue
				}
				mv.AddFloat64(f)
			}
			continue
		}

		// New metric name, make sure it's not disallowed.
		switch metricName {
		case "success", "total", "latency":
			p.l.Warningf("Metric name (%s) in the output conflicts with standard metrics: (success,total,latency). Ignoring.", metricName)
			continue
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			p.l.Warningf("Unsupported value in probe payload for new metric name %s (expected float64): %s", metricName, val)
			continue
		}
		em.AddMetric(metricName, metrics.NewFloat(f))
	}
	return em
}
