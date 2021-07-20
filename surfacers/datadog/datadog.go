// Copyright 2021 The Cloudprober Authors.
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
Package datadog implements a surfacer to export metrics to Datadog.
*/
package datadog

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"google3/third_party/golang/datadog_api_client/api/v1/datadog/datadog"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/surfacers/common/options"
	configpb "github.com/google/cloudprober/surfacers/datadog/proto"
	"google.golang.org/protobuf/proto"
)

/*
	The datadog surfacer presents EventMetrics to the datadog SubmitMetrics APIs,
	using the config passed in.

	Some EventMetrics are not supported here, as the datadog SubmitMetrics API only
	supports float64 type values as the metric value.
*/

// Datadog API limit for metrics included in a SubmitMetrics call
const datadogMaxSeries int = 20

var datadogKind = map[metrics.Kind]string{
	metrics.GAUGE:      "gauge",
	metrics.CUMULATIVE: "count",
}

// DDSurfacer implements a datadog surfacer for datadog metrics.
type DDSurfacer struct {
	c                 *configpb.SurfacerConf
	writeChan         chan *metrics.EventMetrics
	client            *datadog.APIClient
	l                 *logger.Logger
	ignoreLabelsRegex *regexp.Regexp
	prefix            string
	// A cache of []*datadog.Series, used for batch writing to datadog
	ddSeriesCache []datadog.Series
}

func (dd *DDSurfacer) receiveMetricsFromEvent(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			dd.l.Infof("Context canceled, stopping the surfacer write loop")
			return
		case em := <-dd.writeChan:
			dd.recordEventMetrics(ctx, em)
		}
	}
}

func (dd *DDSurfacer) recordEventMetrics(ctx context.Context, em *metrics.EventMetrics) {
	for _, metricKey := range em.MetricsKeys() {
		switch value := em.Metric(metricKey).(type) {
		case metrics.NumValue:
			dd.publishMetrics(ctx, dd.newDDSeries(metricKey, value.Float64(), emLabelsToTags(em), em.Timestamp, em.Kind))
		case *metrics.Map:
			var series []datadog.Series
			for _, k := range value.Keys() {
				tags := emLabelsToTags(em)
				tags = append(tags, fmt.Sprintf("%s:%s", value.MapName, k))
				series = append(series, dd.newDDSeries(metricKey, value.GetKey(k).Float64(), tags, em.Timestamp, em.Kind))
			}
			dd.publishMetrics(ctx, series...)
		case *metrics.Distribution:
			dd.publishMetrics(ctx, dd.distToDDSeries(value.Data(), metricKey, emLabelsToTags(em), em.Timestamp, em.Kind)...)
		}
	}
}

// publish the metrics to datadog, buffering as necessary
func (dd *DDSurfacer) publishMetrics(ctx context.Context, series ...datadog.Series) {
	if len(dd.ddSeriesCache) >= datadogMaxSeries {
		body := *datadog.NewMetricsPayload(dd.ddSeriesCache)
		_, r, err := dd.client.MetricsApi.SubmitMetrics(ctx, body)

		if err != nil {
			dd.l.Errorf("Failed to publish %d series to datadog: %v. Full response: %v", len(dd.ddSeriesCache), err, r)
		}

		dd.ddSeriesCache = dd.ddSeriesCache[:0]
	}

	dd.ddSeriesCache = append(dd.ddSeriesCache, series...)
}

// Create a new datadog series using the values passed in.
func (dd *DDSurfacer) newDDSeries(metricName string, value float64, tags []string, timestamp time.Time, kind metrics.Kind) datadog.Series {
	return datadog.Series{
		Metric: dd.prefix + metricName,
		Points: [][]float64{[]float64{float64(timestamp.Unix()), value}},
		Tags:   &tags,
		Type:   proto.String(datadogKind[kind]),
	}
}

// Take metric labels from an event metric and parse them into a Datadog Dimension struct.
func emLabelsToTags(em *metrics.EventMetrics) []string {
	tags := []string{}

	for _, k := range em.LabelsKeys() {
		tags = append(tags, fmt.Sprintf("%s:%s", k, em.Label(k)))
	}

	return tags
}

func (dd *DDSurfacer) distToDDSeries(d *metrics.DistributionData, metricName string, tags []string, t time.Time, kind metrics.Kind) []datadog.Series {
	ret := []datadog.Series{
		datadog.Series{
			Metric: dd.prefix + metricName + ".sum",
			Points: [][]float64{[]float64{float64(t.Unix()), d.Sum}},
			Tags:   &tags,
			Type:   proto.String(datadogKind[kind]),
		}, {
			Metric: dd.prefix + metricName + ".count",
			Points: [][]float64{[]float64{float64(t.Unix()), float64(d.Count)}},
			Tags:   &tags,
			Type:   proto.String(datadogKind[kind]),
		},
	}

	// Add one point at the value of the Lower Bound per count in the bucket. Each point represents the
	// minimum poissible value that it could have been.
	var points [][]float64
	for i := range d.LowerBounds {
		for n := 0; n < int(d.BucketCounts[i]); n++ {
			points = append(points, []float64{float64(t.Unix()), d.LowerBounds[i]})
		}
	}

	ret = append(ret, datadog.Series{Metric: dd.prefix + metricName, Points: points, Tags: &tags, Type: proto.String(datadogKind[kind])})
	return ret
}

// New creates a new instance of a datadog surfacer, based on the config passed in. It then hands off
// to a goroutine to surface metrics to datadog across a buffered channel.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*DDSurfacer, error) {
	if config.GetApiKey() != "" {
		os.Setenv("DD_API_KEY", config.GetApiKey())
	}
	if config.GetAppKey() != "" {
		os.Setenv("DD_APP_KEY", config.GetAppKey())
	}

	ctx = datadog.NewDefaultContext(ctx)
	configuration := datadog.NewConfiguration()

	client := datadog.NewAPIClient(configuration)

	p := config.GetPrefix()
	if p[len(p)-1] != '.' {
		p += "."
	}

	dd := &DDSurfacer{
		c:         config,
		writeChan: make(chan *metrics.EventMetrics, opts.MetricsBufferSize),
		client:    client,
		l:         l,
		prefix:    p,
	}

	// Set the capacity of this slice to the max metric value, to avoid having to grow the slice.
	dd.ddSeriesCache = make([]datadog.Series, datadogMaxSeries)

	go dd.receiveMetricsFromEvent(ctx)

	dd.l.Info("Initialised Datadog surfacer")
	return dd, nil
}

// Write is a function defined to comply with the surfacer interface, and enables the
// datadog surfacer to receive EventMetrics over the buffered channel.
func (dd *DDSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case dd.writeChan <- em:
	default:
		dd.l.Error("Surfacer's write channel is full, dropping new data.")
	}
}
