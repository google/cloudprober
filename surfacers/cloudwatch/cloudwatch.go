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
Package cloudwatch implements a surfacer to export metrics to AWS Cloudwatch.
*/
package cloudwatch

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/sysvars"

	configpb "github.com/google/cloudprober/surfacers/cloudwatch/proto"
	"github.com/google/cloudprober/surfacers/common/options"
)

// Cloudwatch API limit for metrics included in a PutMetricData call
const maxMetricDatums int = 20

// The dimension named used to identify distributions
const distributionDimensionName string = "le"

// CWSurfacer implements AWS Cloudwatch surfacer.
type CWSurfacer struct {
	c         *configpb.SurfacerConf
	opts      *options.Options
	writeChan chan *metrics.EventMetrics
	session   *cloudwatch.CloudWatch
	l         *logger.Logger

	// A cache of []*cloudwatch.MetricDatum's, used for batch writing to the
	// cloudwatch api.
	cwMetricDatumCache []*cloudwatch.MetricDatum
}

func (cw *CWSurfacer) processIncomingMetrics(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			cw.l.Infof("Context canceled, stopping the surfacer write loop")
			return
		case em := <-cw.writeChan:
			cw.recordEventMetrics(em)
		}
	}
}

// recordEventMetrics takes an EventMetric, which can contain multiple metrics
// of varying types, and loops through each metric in the EventMetric, parsing
// each metric into a structure that is supported by Cloudwatch
func (cw *CWSurfacer) recordEventMetrics(em *metrics.EventMetrics) {
	for _, metricKey := range em.MetricsKeys() {
		if !cw.opts.AllowMetric(metricKey) {
			continue
		}

		switch value := em.Metric(metricKey).(type) {
		case metrics.NumValue:
			cw.publishMetrics(cw.newCWMetricDatum(metricKey, value.Float64(), emLabelsToDimensions(em), em.Timestamp, em.LatencyUnit))

		case *metrics.Map:
			for _, mapKey := range value.Keys() {
				dimensions := emLabelsToDimensions(em)
				dimensions = append(dimensions, &cloudwatch.Dimension{
					Name:  aws.String(value.MapName),
					Value: aws.String(mapKey),
				})
				cw.publishMetrics(cw.newCWMetricDatum(metricKey, value.GetKey(mapKey).Float64(), dimensions, em.Timestamp, em.LatencyUnit))
			}

		case *metrics.Distribution:
			for i, distributionBound := range value.Data().LowerBounds {
				dimensions := append(emLabelsToDimensions(em), &cloudwatch.Dimension{
					Name:  aws.String(distributionDimensionName),
					Value: aws.String(strconv.FormatFloat(distributionBound, 'f', -1, 64)),
				})

				cw.publishMetrics(cw.newCWMetricDatum(metricKey, float64(value.Data().BucketCounts[i]), dimensions, em.Timestamp, em.LatencyUnit))
			}
		}
	}
}

// Publish the metrics to cloudwatch, using the namespace provided from
// configuration.
func (cw *CWSurfacer) publishMetrics(md *cloudwatch.MetricDatum) {
	if len(cw.cwMetricDatumCache) >= maxMetricDatums {
		_, err := cw.session.PutMetricData(&cloudwatch.PutMetricDataInput{
			Namespace:  aws.String(cw.c.GetNamespace()),
			MetricData: cw.cwMetricDatumCache,
		})

		if err != nil {
			cw.l.Errorf("Failed to publish metrics to cloudwatch: %s", err)
		}

		cw.cwMetricDatumCache = cw.cwMetricDatumCache[:0]
	}

	cw.cwMetricDatumCache = append(cw.cwMetricDatumCache, md)
}

// Create a new cloudwatch metriddatum using the values passed in.
func (cw *CWSurfacer) newCWMetricDatum(metricname string, value float64, dimensions []*cloudwatch.Dimension, timestamp time.Time, latencyUnit time.Duration) *cloudwatch.MetricDatum {
	// define the metric datum with default values
	metricDatum := cloudwatch.MetricDatum{
		Dimensions:        dimensions,
		MetricName:        aws.String(metricname),
		Value:             aws.Float64(value),
		StorageResolution: aws.Int64(cw.c.GetResolution()),
		Timestamp:         aws.Time(timestamp),
		Unit:              aws.String(cloudwatch.StandardUnitCount),
	}

	// the cloudwatch api will throw warnings when a timeseries has multiple
	// units, to avoid this always calculate the value as milliseconds.
	if metricname == "latency" {
		metricDatum.Unit = aws.String(cloudwatch.StandardUnitMilliseconds)
		metricDatum.Value = aws.Float64(value * float64(latencyUnit) / float64(time.Millisecond))
	}

	return &metricDatum
}

// Take metric labels from an event metric and parse them into a Cloudwatch Dimension struct.
func emLabelsToDimensions(em *metrics.EventMetrics) []*cloudwatch.Dimension {
	dimensions := []*cloudwatch.Dimension{}

	for _, k := range em.LabelsKeys() {
		dimensions = append(dimensions, &cloudwatch.Dimension{
			Name:  aws.String(k),
			Value: aws.String(em.Label(k)),
		})
	}

	return dimensions
}

// getRegion provides an order of precedence for the lookup of the AWS Region
// Precendence:
// 1. The region passed in via config
// 2. The region discovered from the metadata endpoint
// 3. By returning nil, the environment variable AWS_REGION as evaluated by the
// the AWS SDK.
func getRegion(config *configpb.SurfacerConf) string {
	if config.Region != nil {
		return config.GetRegion()
	}

	vars := sysvars.Vars()
	if v, exists := vars["EC2_Region"]; exists {
		return v
	}

	return ""
}

// New creates a new instance of a cloudwatch surfacer, based on the config
// passed in. It then hands off to a goroutine to surface metrics to cloudwatch
// across a buffered channel.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*CWSurfacer, error) {
	region := getRegion(config)

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: &region,
		},
	}))

	cw := &CWSurfacer{
		c:         config,
		opts:      opts,
		writeChan: make(chan *metrics.EventMetrics, opts.MetricsBufferSize),
		session:   cloudwatch.New(sess),
		l:         l,
	}

	// Set the capacity of this slice to the max metric value, to avoid having to
	// grow the slice.
	cw.cwMetricDatumCache = make([]*cloudwatch.MetricDatum, 0, maxMetricDatums)

	go cw.processIncomingMetrics(ctx)

	cw.l.Info("Initialised Cloudwatch surfacer")
	return cw, nil
}

// Write is a function defined to comply with the surfacer interface, and enables the
// cloudwatch surfacer to receive EventMetrics over the buffered channel.
func (cw *CWSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case cw.writeChan <- em:
	default:
		cw.l.Error("Surfacer's write channel is full, dropping new data.")
	}
}
