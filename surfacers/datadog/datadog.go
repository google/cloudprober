package datadog

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	datadog "github.com/DataDog/datadog-api-client-go/api/v1/datadog"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	configpb "github.com/google/cloudprober/surfacers/datadog/proto"
	"google.golang.org/protobuf/proto"
)

/*
	The datadog surfacer presents EventMetrics to the cloudwatch putmetricdata APIs,
	using the config passed in.

	Some EventMetrics are not supported here, as the cloudwatch putmetricdata API only
	supports float64 type values as the metric value.
*/

// Cloudwatch API limit for metrics included in a PutMetricData call
const cloudwatchMaxMetricDatums int = 20

// The dimension named used to identify distributions
const distributionDimensionName string = "le"

//
const gauge string = "string"

// DDSurfacer implements a cloudwatch surfacer for cloudwatch metrics.
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
			dd.emToDDSeries(ctx, em)
		}
	}
}

func (dd *DDSurfacer) emToDDSeries(ctx context.Context, em *metrics.EventMetrics) {
	for _, metricKey := range em.MetricsKeys() {
		switch value := em.Metric(metricKey).(type) {
		case metrics.NumValue:
			dd.publishMetrics(ctx, dd.newDDSeries(metricKey, value.Float64(), emLabelsToTags(em), em.Timestamp))
		case *metrics.Map:
			for _, k := range value.Keys() {
				tags := emLabelsToTags(em)
				tags = append(tags, fmt.Sprintf("%s:%s", value.MapName, k))
				dd.publishMetrics(ctx, dd.newDDSeries(metricKey, value.GetKey(k).Float64(), tags, em.Timestamp))
			}
		case *metrics.Distribution:
			for _, series := range dd.distToDDSeries(value.Data(), metricKey, emLabelsToTags(em), em.Timestamp) {
				dd.publishMetrics(ctx, series)
			}
			//dd.publishMetrics(ctx, distToDDSeries(value, metricKey, emLabelsToTags(em), em.Timestamp))
		}
	}
}

// publish the metrics to cloudwatch, using the namespace provided from configuration
func (dd *DDSurfacer) publishMetrics(ctx context.Context, series datadog.Series) {
	if len(dd.ddSeriesCache) >= 20 {
		body := *datadog.NewMetricsPayload(dd.ddSeriesCache)
		_, r, err := dd.client.MetricsApi.SubmitMetrics(ctx, body)

		if err != nil {
			dd.l.Errorf("Failed to publish metrics to datadog: %v. Full response: %v", err, r)
		}

		dd.ddSeriesCache = dd.ddSeriesCache[:0]
	}

	dd.ddSeriesCache = append(dd.ddSeriesCache, series)
}

// Create a new cloudwatch metriddatum using the values passed in. The value for a metric in cloudwatch must be of float64 type.
func (dd *DDSurfacer) newDDSeries(metricName string, value float64, tags []string, timestamp time.Time) datadog.Series {
	return datadog.Series{
		Metric: dd.prefix + metricName,
		Points: [][]float64{[]float64{float64(timestamp.Unix()), value}},
		Tags:   &tags,
		Type:   proto.String(gauge),
	}
}

// Take metric labels from an event metric and parse them into a Cloudwatch Dimension struct.
func emLabelsToTags(em *metrics.EventMetrics) []string {
	var tags []string

	for _, k := range em.LabelsKeys() {
		tags = append(tags, fmt.Sprintf("%s:%s", k, em.Label(k)))
	}

	return tags
}

func (dd *DDSurfacer) distToDDSeries(d *metrics.DistributionData, metricName string, tags []string, t time.Time) []datadog.Series {
	ret := []datadog.Series{
		datadog.Series{
			Metric: dd.prefix + metricName + ".sum",
			Points: [][]float64{[]float64{float64(t.Unix()), d.Sum}},
			Tags:   &tags,
			Type:   proto.String(gauge),
		}, {
			Metric: dd.prefix + metricName + ".count",
			Points: [][]float64{[]float64{float64(t.Unix()), float64(d.Count)}},
			Tags:   &tags,
			Type:   proto.String(gauge),
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

	ret = append(ret, datadog.Series{Metric: dd.prefix + metricName, Points: points, Tags: &tags, Type: proto.String(gauge)})
	return ret
}

// New creates a new instance of a cloudwatch surfacer, based on the config passed in. It then hands off
// to a goroutine to surface metrics to cloudwatch across a buffered channel.
func New(ctx context.Context, config *configpb.SurfacerConf, l *logger.Logger) (*DDSurfacer, error) {

	l.Infof("1\n")
	os.Setenv("DD_APP_KEY", config.GetAppKey())
	os.Setenv("DD_API_KEY", config.GetApiKey())

	l.Infof("2\n")
	ctx = datadog.NewDefaultContext(ctx)
	configuration := datadog.NewConfiguration()
	l.Infof("3\n")

	client := datadog.NewAPIClient(configuration)
	l.Infof("4\n")

	p := config.GetPrefix()
	if p[len(p)-1] != '.' {
		p += "."
	}

	dd := &DDSurfacer{
		c:         config,
		writeChan: make(chan *metrics.EventMetrics, config.GetMetricsBufferSize()),
		client:    client,
		l:         l,
		prefix:    p,
	}
	l.Infof("5\n")

	if config.GetIgnoreProberTypes() != "" {
		r, err := regexp.Compile(config.GetIgnoreProberTypes())
		if err != nil {
			return nil, err
		}
		dd.ignoreLabelsRegex = r
	}

	// Set the capacity of this slice to the max metric value, to avoid having to grow the slice.
	dd.ddSeriesCache = make([]datadog.Series, 0)

	l.Infof("6\n")
	go dd.receiveMetricsFromEvent(ctx)

	dd.l.Info("Initialised Cloudwatch surfacer")
	return dd, nil
}

// Write is a function defined to comply with the surfacer interface, and enables the
// cloudwatch surfacer to receive EventMetrics over the buffered channel.
func (dd *DDSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case dd.writeChan <- em:
	default:
		dd.l.Error("Surfacer's write channel is full, dropping new data.")
	}
}
