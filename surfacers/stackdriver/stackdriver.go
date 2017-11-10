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
Package stackdriver implements the Stackdriver version of the Surfacer
object. This package allows users to create an initialized Stack Driver
Surfacer and use it to write custom metrics data.
*/
package stackdriver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/google/cloudprober/logger"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/monitoring/v3"

	"github.com/google/cloudprober/metrics"
)

//-----------------------------------------------------------------------------
// Stack Driver Surfacer Specific Code
//-----------------------------------------------------------------------------

// SDSurfacer structure for StackDriver, which includes an authenticated client
// for making StackDriver API calls, and a registered which is in charge of
// keeping track of what metrics have already been registereded
type SDSurfacer struct {

	// Configuration
	c *SurfacerConf

	// Internal cache for saving metric data until a batch is sent
	cache map[string]*monitoring.TimeSeries

	// Channel for writing the data without blocking
	writeChan chan *metrics.EventMetrics

	// VM Information
	projectName  string
	instanceName string
	zone         string

	// Time when stackdriver module was initialized. This is used as start time
	// for cumulative metrics.
	startTime time.Time

	// Cloud logger
	l       *logger.Logger
	failCnt int64

	// Monitoring client
	client *monitoring.Service
}

// New initializes a SDSurfacer for Stack Driver with all its necessary internal
// variables for call references (project and instances variables) as well
// as provisioning it with clients for making the necessary API calls. New
// requires you to pass in a valid stackdriver surfacer configuration.
func New(config *SurfacerConf, l *logger.Logger) (*SDSurfacer, error) {
	// Create a cache, which is used for batching write requests together,
	// and a channel for writing data.
	s := SDSurfacer{
		cache:     make(map[string]*monitoring.TimeSeries),
		writeChan: make(chan *metrics.EventMetrics, 1000),
		c:         config,
		startTime: time.Now(),
		l:         l,
	}

	// TODO: Validate that the config has all the necessary
	// values

	// Find all the necessary information for writing metrics to Stack
	// Driver.
	var err error
	if s.projectName, err = metadata.ProjectID(); err != nil {
		return nil, fmt.Errorf("unable to retrieve project name: %v", err)
	}

	if s.instanceName, err = metadata.InstanceName(); err != nil {
		return nil, fmt.Errorf("unable to retrieve instance name: %v", err)
	}

	if s.zone, err = metadata.Zone(); err != nil {
		return nil, fmt.Errorf("unable to retrieve instance zone: %v", err)
	}

	// Create monitoring client
	// TODO: Currently we don't make use of the context to timeout the
	// requests, but we should.
	httpClient, err := google.DefaultClient(context.TODO(), monitoring.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	s.client, err = monitoring.New(httpClient)
	if err != nil {
		return nil, err
	}

	// Start either the writeAsync or the writeBatch, depending on if we are
	// batching or not.
	go func() {
		if s.c.GetBatch() {
			s.writeBatch()
		} else {
			s.writeAsync()
		}
	}()

	s.l.Info("Create a new stackdriver surfacer")
	return &s, nil
}

// Write queues a message to be written to stackdriver.
func (s *SDSurfacer) Write(ctxIn context.Context, em *metrics.EventMetrics) {
	// Write inserts the data to be written into channel. This channel is
	// watched by writeAsync or writeBatch and will make the necessary calls
	// to the Stackdriver API to write the data from the channel.
	select {
	case s.writeChan <- em:
	default:
		s.l.Warningf("SDSurfacer's write channel is full, dropping new data.")
	}
}

// writeAsync polls the writeChan waiting for a new write packet to be added.
// When a packet is put on the channel, writeAsync creates a client based on the
// passed in context, creates a time series create call for the data that is
// input, and sends that time series create call to stackdriver to write a new
// custom metric using the given context.
//
// writeAsync is set up to run as an infinite goroutine call in the New function
// to allow it to write asynchronously to Stack Driver
func (s *SDSurfacer) writeAsync() {
	for em := range s.writeChan {
		// Now that we've created the new metric, we can write the data. Making
		// a time series create call will automatically register a new metric
		// with the correct information if it does not already exist.
		//	Ref: https://cloud.google.com/monitoring/custom-metrics/creating-metrics#auto-creation
		ts := s.recordEventMetrics(em)
		requestBody := monitoring.CreateTimeSeriesRequest{
			TimeSeries: ts,
		}
		_, err := s.client.Projects.TimeSeries.Create("projects/"+s.projectName, &requestBody).Do()
		if err != nil {
			s.failCnt++
			s.l.Warningf("Unable to fulfill TimeSeries Create call. Err: %v", err)
			return
		}
	}
}

// writeBatch polls the writeChan and the sendChan waiting for either a new
// write packet or a new context. If data comes in on the writeChan, then
// the data is pulled off and put into the cache (if there is already an
// entry into the cache for the same metric, it updates the metric to the
// new data). If ticker fires, then the metrics in the cache
// are batched together. The Stackdriver API has a limit on the maximum number
// of metrics that can be sent in a single request, so we may have to make
// multiple requests to the Stackdriver API to send the full cache of metrics.
//
// writeBatch is set up to run as an infinite goroutine call in the New function
// to allow it to write asynchronously to Stack Driver.
func (s *SDSurfacer) writeBatch() {
	batchTicker := time.Tick(time.Duration(s.c.GetBatchTimerSec()) * time.Second)
	for {
		select {
		case em := <-s.writeChan:
			// Process EventMetrics to build timeseries using them and cache the timeseries
			// objects.
			s.recordEventMetrics(em)
		case <-batchTicker:
			// Empty time series writes cause an error to be returned, so
			// we skip any calls that write but wouldn't set any data.
			if len(s.cache) == 0 {
				break
			}

			var ts []*monitoring.TimeSeries
			for _, v := range s.cache {
				ts = append(ts, v)
			}

			// We batch the time series into appropriately-sized sets
			// and write them
			for i := 0; i < len(ts); i += int(s.c.GetBatchSize()) {
				endIndex := min(len(ts), i+int(s.c.GetBatchSize()))

				s.l.Infof("Sending entries %d through %d of %d", i, endIndex, len(ts))

				// Now that we've created the new metric, we can write the data. Making
				// a time series create call will automatically register a new metric
				// with the correct information if it does not already exist.
				// Ref: https://cloud.google.com/monitoring/custom-metrics/creating-metrics#auto-creation
				requestBody := monitoring.CreateTimeSeriesRequest{
					TimeSeries: ts[i:endIndex],
				}
				if _, err := s.client.Projects.TimeSeries.Create("projects/"+s.projectName, &requestBody).Do(); err != nil {
					s.failCnt++
					s.l.Warningf("Unable to fulfill TimeSeries Create call. Err: %v", err)
				}
			}

			// Flush the cache after we've finished writing so we don't accidentally
			// re-write metric values that haven't been written over several write
			// cycles.
			for k := range s.cache {
				delete(s.cache, k)
			}
		}
	}

}

//-----------------------------------------------------------------------------
// StackDriver Object Creation and Helper Functions
//-----------------------------------------------------------------------------

// recordTimeSeries forms a timeseries object from the given arugments, records
// it in the cache if batch processing is enabled, and returns it.
//
// More information on the object and specific fields can be found here:
//	https://cloud.google.com/monitoring/api/ref_v3/rest/v3/TimeSeries
func (s *SDSurfacer) recordTimeSeries(metricKind, metricName, msgType string, labels map[string]string, timestamp time.Time, tv *monitoring.TypedValue, cacheKey string) *monitoring.TimeSeries {
	startTime := s.startTime.Format(time.RFC3339Nano)
	if metricKind == "GAUGE" {
		startTime = timestamp.Format(time.RFC3339Nano)
	}
	ts := &monitoring.TimeSeries{
		// The URL address for our custom metric, must match the
		// name we used in the MetricDescriptor.
		Metric: &monitoring.Metric{
			Type:   s.c.GetMonitoringUrl() + metricName,
			Labels: labels,
		},

		// Resource is required only if we want the data to be parsable
		// on the gce-instance level (as opposed to all globally).
		Resource: &monitoring.MonitoredResource{
			Type: "gce_instance",
			Labels: map[string]string{
				"instance_id": s.instanceName,
				"zone":        s.zone,
			},
		},

		// Must match the MetricKind and ValueType of the MetricDescriptor.
		MetricKind: metricKind,
		ValueType:  msgType,

		// Create a single data point, this could be utilized to create
		// a batch of points instead of a single point if the write
		// rate is too high.
		Points: []*monitoring.Point{
			{
				Interval: &monitoring.TimeInterval{
					StartTime: startTime,
					EndTime:   timestamp.Format(time.RFC3339Nano),
				},
				Value: tv,
			},
		},
	}

	if s.c.GetBatch() && s.cache != nil {
		// We create a key that is a composite of both the name and the
		// labels so we can make sure that the cache holds all distinct
		// values and not just the ones with different names.
		s.cache[metricName+","+cacheKey] = ts
	}
	return ts

}

// sdKind converts EventMetrics kind to StackDriver kind string.
func (s *SDSurfacer) sdKind(kind metrics.Kind) string {
	switch kind {
	case metrics.GAUGE:
		return "GAUGE"
	case metrics.CUMULATIVE:
		return "CUMULATIVE"
	default:
		return ""
	}
}

// processLabels processes EventMetrics labels to generate:
//	- a map of label key values to use in StackDriver timeseries,
//	- a labels key of the form label1_key=label1_val,label2_key=label2_val,
//	  used for caching.
//	- prefix for metric names, usually <ptype>/<probe>.
func processLabels(em *metrics.EventMetrics) (labels map[string]string, labelsKey, metricPrefix string) {
	labels = make(map[string]string)
	var sortedLabels []string // we use this for cache key below
	var ptype, probe string
	for _, k := range em.LabelsKeys() {
		if k == "ptype" {
			ptype = em.Label(k)
			continue
		}
		if k == "probe" {
			probe = em.Label(k)
			continue
		}
		labels[k] = em.Label(k)
		sortedLabels = append(sortedLabels, k+"="+labels[k])
	}
	labelsKey = strings.Join(sortedLabels, ",")

	if ptype != "" {
		metricPrefix += ptype + "/"
	}
	if probe != "" {
		metricPrefix += probe + "/"
	}
	return
}

// recordEventMetrics processes the incoming EventMetrics objects and builds
// TimeSeries from it.
//
// Since stackdriver doesn't support metrics.String and metrics.Map value types,
// it converts them to a numerical types (stackdriver type Double) with
// additional labels. See the inline comments for this conversion is done.
func (s *SDSurfacer) recordEventMetrics(em *metrics.EventMetrics) (ts []*monitoring.TimeSeries) {
	metricKind := s.sdKind(em.Kind)
	if metricKind == "" {
		s.l.Warningf("Unknown event metrics type (not CUMULATIVE or GAUGE): %v", em.Kind)
		return
	}

	emLabels, cacheKey, metricPrefix := processLabels(em)

	for _, k := range em.MetricsKeys() {
		// Create a copy of emLabels for use in timeseries object.
		mLabels := make(map[string]string)
		for k, v := range emLabels {
			mLabels[k] = v
		}
		name := metricPrefix + k

		if !validMetricLength(name, s.c.GetMonitoringUrl()) {
			s.l.Warningf("Message name %q is greater than the 100 character limit, skipping write", name)
			continue
		}

		// Create the correct TimeSeries object based on the incoming data
		val := em.Metric(k)

		// If metric value is of type numerical value.
		if v, ok := val.(metrics.NumValue); ok {
			f := float64(v.Int64())
			ts = append(ts, s.recordTimeSeries(metricKind, name, "DOUBLE", mLabels, em.Timestamp, &monitoring.TypedValue{DoubleValue: &f}, cacheKey))
			continue
		}

		// If metric value is of type String.
		if v, ok := val.(metrics.String); ok {
			// Since StackDriver doesn't support string value type for custom metrics,
			// we convert string metrics into a numeric metric with an additional label
			// val="string-val".
			//
			// metrics.String stringer wraps string values in a single "". Remove those
			// for stackdriver.
			mLabels["val"] = strings.Trim(v.String(), "\"")
			f := float64(1)
			ts = append(ts, s.recordTimeSeries(metricKind, name, "DOUBLE", mLabels, em.Timestamp, &monitoring.TypedValue{DoubleValue: &f}, cacheKey))
			continue
		}

		// If metric value is of type Map.
		if mapValue, ok := val.(*metrics.Map); ok {
			// Since StackDriver doesn't support Map value type, we convert Map values
			// to multiple timeseries with map's KeyName and key as labels.
			for _, mapKey := range mapValue.Keys() {
				mmLabels := make(map[string]string)
				for lk, lv := range mLabels {
					mmLabels[lk] = lv
				}
				mmLabels[mapValue.MapName] = mapKey
				f := float64(mapValue.GetKey(mapKey).Int64())
				ts = append(ts, s.recordTimeSeries(metricKind, name, "DOUBLE", mmLabels, em.Timestamp, &monitoring.TypedValue{DoubleValue: &f}, cacheKey))
			}
			continue
		}

		// If metric value is of type Distribution.
		if distValue, ok := val.(*metrics.Distribution); ok {
			ts = append(ts, s.recordTimeSeries(metricKind, name, "DISTRIBUTION", mLabels, em.Timestamp, distValue.StackdriverTypedValue(), cacheKey))
			continue
		}

		// We'll reach here only if encounter an unsupported value type.
		s.l.Warningf("Unsupported value type: %v", val)
	}
	return ts
}

//-----------------------------------------------------------------------------
// Non-stackdriver Helper Functions
//-----------------------------------------------------------------------------

// checkMetricLength checks if the combination of the metricName and the url
// prefix are longer than 100 characters, which is illegal in a Stackdriver
// call. Stack Driver doesn't allow custom metrics with more than 100 character
// names, so we have a check to see if we are going over the limit.
//	Ref: https://cloud.google.com/monitoring/api/v3/metrics#metric_names
func validMetricLength(metricName string, monitoringURL string) bool {
	if len(metricName)+len(monitoringURL) > 100 {
		return false
	}
	return true
}

// Function to return the min of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
