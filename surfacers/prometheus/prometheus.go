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
Package prometheus provides a prometheus surfacer for Cloudprober. Prometheus
surfacer exports incoming metrics over a web interface in a format that
prometheus understands (http://prometheus.io).

This surfacer processes each incoming EventMetrics and holds the latest value
and timestamp for each metric in memory. These metrics are made available
through a web URL (default: /metrics), which Prometheus scrapes at a regular
interval.

Example /metrics page:
#TYPE sent counter
sent{ptype="dns",probe="vm-to-public-dns",dst="8.8.8.8"} 181299 1497330037000
sent{ptype="ping",probe="vm-to-public-dns",dst="8.8.4.4"} 362600 1497330037000
#TYPE rcvd counter
rcvd{ptype="dns",probe="vm-to-public-dns",dst="8.8.8.8"} 181234 1497330037000
rcvd{ptype="ping",probe="vm-to-public-dns",dst="8.8.4.4"} 362600 1497330037000
*/
package prometheus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

// Prometheus metric and label names should match the following regular
// expressions. Since, "-" is commonly used in metric and label names, we
// replace it by "_". If a name still doesn't match the regular expression, we
// ignore it with a warning log message.
const (
	ValidMetricNameRegex = "^[a-zA-Z_:]([a-zA-Z0-9_:])*$"
	ValidLabelNameRegex  = "^[a-zA-Z_]([a-zA-Z0-9_])*$"
)

// queriesQueueSize defines how many queries can we queue before we start
// blocking on previous queries to finish.
const queriesQueueSize = 10

var (
	// Cache of EventMetric label to prometheus label mapping. We use it to
	// quickly lookup if we have already seen a label and we have a prometheus
	// label corresponding to it.
	promLabelNames = make(map[string]string)

	// Cache of EventMetric metric to prometheus metric mapping. We use it to
	// quickly lookup if we have already seen a metric and we have a prometheus
	// metric name corresponding to it.
	promMetricNames = make(map[string]string)
)

type promMetric struct {
	typ      string
	data     map[string]*dataPoint
	dataKeys []string // To keep data keys ordered
}

type dataPoint struct {
	value     string
	timestamp int64
}

// PromSurfacer implements a prometheus surfacer for Cloudprober. PromSurfacer
// organizes metrics into a two-level data structure:
//		1. Metric name -> PromMetric data structure dict.
//    2. A PromMetric organizes data associated with a metric in a
//			 Data key -> Data point map, where data point consists of a value
//       and timestamp.
// Data key represents a unique combination of metric name and labels.
type PromSurfacer struct {
	c           *SurfacerConf              // Configuration
	emChan      chan *metrics.EventMetrics // Buffered channel to store incoming EventMetrics
	metrics     map[string]*promMetric     // Metric name to promMetric mapping
	metricNames []string                   // Metric names, to keep names ordered.
	queryChan   chan http.ResponseWriter   // Query channel
	l           *logger.Logger

	// Regexes for metric and label names.
	metricNameRe *regexp.Regexp
	labelNameRe  *regexp.Regexp
}

// New returns a prometheus surfacer based on the config provided. It sets up a
// goroutine to process both the incoming EventMetrics and the web requests for
// the URL handler /metrics.
func New(ctx context.Context, name string, config *SurfacerConf) (*PromSurfacer, error) {
	if config == nil {
		config = &SurfacerConf{}
	}
	ps := &PromSurfacer{
		c:            config,
		emChan:       make(chan *metrics.EventMetrics, config.GetMetricsBufferSize()),
		queryChan:    make(chan http.ResponseWriter, queriesQueueSize),
		metrics:      make(map[string]*promMetric),
		metricNameRe: regexp.MustCompile(ValidMetricNameRegex),
		labelNameRe:  regexp.MustCompile(ValidLabelNameRegex),
	}

	var err error
	// Create a new cloud logger specifically for this project and instance
	ps.l, err = logger.New(ctx, name, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create cloud logger for surfacer %s: %v", name, err)
	}

	done := make(chan interface{}, 1)

	// Start a goroutine to process the incoming EventMetrics as well as
	// the incoming web queries. To avoid data access race conditions, we do
	// one thing at a time.
	go func() {
		for {
			select {
			case em := <-ps.emChan:
				ps.record(em)
			case w := <-ps.queryChan:
				ps.writeData(w)
				done <- true
			}
		}
	}()

	http.HandleFunc(ps.c.GetMetricsUrl(), func(w http.ResponseWriter, r *http.Request) {
		ps.queryChan <- w
		<-done
	})

	// Start web server in a goroutine.
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", ps.c.GetPort()), nil)
		glog.Exit("Prometheus surfacer web server exited.")
	}()

	glog.Infof("Initialized prometheus exporter at %d/%s", ps.c.GetPort(), ps.c.GetMetricsUrl())
	return ps, nil
}

// Write simply queues the EventMetrics for later processing.
func (ps *PromSurfacer) Write(ctxIn context.Context, em *metrics.EventMetrics) {
	ps.emChan <- em
}

func promType(em *metrics.EventMetrics) string {
	switch em.Kind {
	case metrics.CUMULATIVE:
		return "counter"
	case metrics.GAUGE:
		return "gauge"
	default:
		return "unknown"
	}
}

// promTime converts time.Time to Unix milliseconds.
func promTime(t time.Time) int64 {
	return t.UnixNano() / (1000 * 1000)
}

func (ps *PromSurfacer) recordMetric(metricName string, labels []string, value string, em *metrics.EventMetrics) {
	key := metricName + "{" + strings.Join(labels, ",") + "}"
	// Recognized metric
	if pm := ps.metrics[metricName]; pm != nil {
		// Recognized metric name and labels combination.
		if pm.data[key] != nil {
			pm.data[key].value = value
			pm.data[key].timestamp = promTime(em.Timestamp)
			return
		}
		pm.data[key] = &dataPoint{
			value:     value,
			timestamp: promTime(em.Timestamp),
		}
		pm.dataKeys = append(pm.dataKeys, key)
	} else {
		// Newly discovered metric name.
		ps.metrics[metricName] = &promMetric{
			typ: promType(em),
			data: map[string]*dataPoint{
				key: &dataPoint{
					value:     value,
					timestamp: promTime(em.Timestamp),
				},
			},
			dataKeys: []string{key},
		}
		ps.metricNames = append(ps.metricNames, metricName)
	}
	return
}

// checkLabelName finds a prometheus label name for an incoming label. If label
// is found to be invalid even after some basic conversions, a zero string is
// returned.
func (ps *PromSurfacer) checkLabelName(k string) string {
	// Before checking with regex, see if this label name is
	// already known. This block will be entered only once per
	// label name.
	if promLabel, ok := promLabelNames[k]; ok {
		return promLabel
	}

	ps.l.Infof("Checking validity of new label: %s", k)
	// We'll come here only once per label name.

	// Prometheus doesn't support "-" in metric names.
	labelName := strings.Replace(k, "-", "_", -1)
	if !ps.labelNameRe.MatchString(labelName) {
		// Explicitly store a zero string so that we don't check it again.
		promLabelNames[k] = ""
		ps.l.Warningf("Ignoring invalid prometheus label name: %s", k)
		return ""
	}
	promLabelNames[k] = labelName
	return labelName
}

// checkMetricName finds a prometheus metric name for an incoming metric. If metric
// is found to be invalid even after some basic conversions, a zero string is
// returned.
func (ps *PromSurfacer) checkMetricName(k string) string {
	// Before checking with regex, see if this metric name is
	// already known. This block will be entered only once per
	// metric name.
	if metricName, ok := promMetricNames[k]; ok {
		return metricName
	}

	ps.l.Infof("Checking validity of new metric: %s", k)
	// We'll come here only once per metric name.

	// Prometheus doesn't support "-" in metric names.
	metricName := strings.Replace(k, "-", "_", -1)
	if !ps.metricNameRe.MatchString(metricName) {
		// Explicitly store a zero string so that we don't check it again.
		promMetricNames[k] = ""
		ps.l.Warningf("Ignoring invalid prometheus metric name: %s", k)
		return ""
	}
	promMetricNames[k] = metricName
	return metricName
}

// record processes the incoming EventMetrics and updates the in-memory
// database.
//
// Since prometheus doesn't support certain metrics.Value types, we handle them
// differently.
//
// metrics.Map value type:  We break Map values into multiple data keys, with
// each map key corresponding to a label in the data key.
// For example, "resp-code map:code 200:45 500:2" gets converted into:
//   resp-code{code=200} 45
//   resp-code{code=500}  2
//
// metrics.String value type: We convert string value type into a data key with
// val="value" label.
// For example, "version cloudprober-20170608-RC00" gets converted into:
//   version{val=cloudprober-20170608-RC00} 1
func (ps *PromSurfacer) record(em *metrics.EventMetrics) {
	var labels []string
	for _, k := range em.LabelsKeys() {
		if labelName := ps.checkLabelName(k); labelName != "" {
			labels = append(labels, labelName+"=\""+em.Label(k)+"\"")
		}
	}

	for _, metricName := range em.MetricsKeys() {
		pMetricName := ps.checkMetricName(metricName)
		if pMetricName == "" {
			// No prometheus metric name found for this metric.
			continue
		}
		val := em.Metric(metricName)

		// Map values get expanded into metrics with extra label.
		if mapVal, ok := val.(*metrics.Map); ok {
			labelName := ps.checkLabelName(mapVal.MapName)
			if labelName == "" {
				continue
			}
			for _, k := range mapVal.Keys() {
				labelsWithMap := append(labels, labelName+"=\""+k+"\"")
				ps.recordMetric(pMetricName, labelsWithMap, mapVal.GetKey(k).String(), em)
			}
			continue
		}
		// String values get converted into a label.
		if _, ok := val.(metrics.String); ok {
			newLabels := append(labels, "val="+val.String())
			ps.recordMetric(pMetricName, newLabels, "1", em)
			continue
		}

		// All other value types, mostly numerical types.
		ps.recordMetric(pMetricName, labels, val.String(), em)
	}
}

// writeData writes metrics data on w io.Writer
func (ps *PromSurfacer) writeData(w io.Writer) {
	for _, name := range ps.metricNames {
		pm := ps.metrics[name]
		fmt.Fprintf(w, "#TYPE %s %s\n", name, pm.typ)
		for _, k := range pm.dataKeys {
			fmt.Fprintf(w, "%s %s %d\n", k, pm.data[k].value, pm.data[k].timestamp)
		}
	}
}
