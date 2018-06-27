// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this postgres except in compliance with the License.
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
Package postgres implements "postgres" surfacer. This surfacer type is in
experimental phase right now.

To use this surfacer, add a stanza similar to the following to your
cloudprober config:

surfacer {
  type: POSTGRES
	postgres_surfacer {
	  connection_string: "postgresql://root:root@localhost/cloudprober?sslmode=disable"
	  metrics_table_name: "metrics"
  }
}
*/
package postgres

import (
	"context"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"

	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/lib/pq"

	configpb "github.com/google/cloudprober/surfacers/postgres/proto"
)

// pgMetric represents a single metric and corresponds to a single row in the
// metrics table.
type pgMetric struct {
	time       time.Time
	metricName string
	value      string
	labels     map[string]string
}

func updateLabelMap(labels map[string]string, extraLabels ...[2]string) map[string]string {
	if len(extraLabels) == 0 {
		return labels
	}
	labelsCopy := make(map[string]string)
	for k, v := range labels {
		labelsCopy[k] = v
	}
	for _, extraLabel := range extraLabels {
		labelsCopy[extraLabel[0]] = extraLabel[1]
	}
	return labelsCopy
}

// labelsJSON takes the labels array and formats it for insertion into
// postgres jsonb labels column, storing each label as k,v json object
func labelsJSON(labels map[string]string) (string, error) {
	bs, err := json.Marshal(labels)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func newPGMetric(t time.Time, metricName, val string, labels map[string]string) pgMetric {
	return pgMetric{
		time:       t,
		metricName: metricName,
		value:      val,
		labels:     labels,
	}
}

func distToPGMetrics(d *metrics.DistributionData, metricName string, labels map[string]string, t time.Time) []pgMetric {
	pgMerics := []pgMetric{
		newPGMetric(t, metricName+"_sum", strconv.FormatFloat(d.Sum, 'f', -1, 64), labels),
		newPGMetric(t, metricName+"_count", strconv.FormatInt(d.Count, 10), labels),
	}

	// Create and format all metrics for each bucket in this distribution. Each
	// bucket is assigned a metric name suffixed with "_bucket" and labeled with
	// the corresponding bucket as "le: {bucket}"
	var val int64
	for i := range d.LowerBounds {
		val += d.BucketCounts[i]
		var lb string
		if i == len(d.LowerBounds)-1 {
			lb = "+Inf"
		} else {
			lb = strconv.FormatFloat(d.LowerBounds[i+1], 'f', -1, 64)
		}
		labelsWithBucket := updateLabelMap(labels, [2]string{"le", lb})
		pgMerics = append(pgMerics, newPGMetric(t, metricName+"_bucket", strconv.FormatInt(val, 10), labelsWithBucket))
	}

	return pgMerics
}

// emToPGMetrics converts an EventMetrics struct into a list of pgMetrics.
func emToPGMetrics(em *metrics.EventMetrics) []pgMetric {
	baseLabels := make(map[string]string)
	for _, k := range em.LabelsKeys() {
		baseLabels[k] = em.Label(k)
	}

	pgMerics := []pgMetric{}
	for _, metricName := range em.MetricsKeys() {
		val := em.Metric(metricName)

		// Map metric
		if mapVal, ok := val.(*metrics.Map); ok {
			for _, k := range mapVal.Keys() {
				labels := updateLabelMap(baseLabels, [2]string{mapVal.MapName, k})
				pgMerics = append(pgMerics, newPGMetric(em.Timestamp, metricName, mapVal.GetKey(k).String(), labels))
			}
			continue
		}

		// Distribution metric
		if distVal, ok := val.(*metrics.Distribution); ok {
			pgMerics = append(pgMerics, distToPGMetrics(distVal.Data(), metricName, baseLabels, em.Timestamp)...)
			continue
		}

		// Convert string metrics to a numeric metric by moving metric value to
		// the "val" label and setting the metric value to 1.
		// For example: version="1.11" becomes version{val="1.11"}=1
		if _, ok := val.(metrics.String); ok {
			labels := updateLabelMap(baseLabels, [2]string{"val", val.String()})
			pgMerics = append(pgMerics, newPGMetric(em.Timestamp, metricName, "1", labels))
			continue
		}

		pgMerics = append(pgMerics, newPGMetric(em.Timestamp, metricName, val.String(), baseLabels))
	}
	return pgMerics
}

// Surfacer structures for writing to postgres.
type Surfacer struct {
	// Configuration
	c *configpb.SurfacerConf

	// Channel for incoming data.
	writeChan chan *metrics.EventMetrics

	// Cloud logger
	l *logger.Logger

	openDB func(connectionString string) (*sql.DB, error)
	db     *sql.DB
}

// New initializes a Postgres surfacer. Postgres surfacer inserts probe results
// into a postgres database.
func New(config *configpb.SurfacerConf, l *logger.Logger) (*Surfacer, error) {
	s := &Surfacer{
		c: config,
		l: l,
		openDB: func(cs string) (*sql.DB, error) {
			return sql.Open("postgres", cs)
		},
	}
	return s, s.init()
}

// writeMetrics parses events metrics into postgres rows, starts a transaction
// and inserts all discreet metric rows represented by the EventMetrics
func (s *Surfacer) writeMetrics(em *metrics.EventMetrics) error {
	// Begin a transaction.
	txn, err := s.db.Begin()
	if err != nil {
		return err
	}

	// Prepare a statement to COPY table from the STDIN.
	stmt, err := txn.Prepare(pq.CopyIn(s.c.GetMetricsTableName(), "time", "metric_name", "value", "labels"))
	if err != nil {
		return err
	}

	for _, pgMetric := range emToPGMetrics(em) {
		var s string
		if s, err = labelsJSON(pgMetric.labels); err != nil {
			return err
		}
		if _, err = stmt.Exec(pgMetric.time, pgMetric.metricName, pgMetric.value, s); err != nil {
			return err
		}
	}

	if _, err = stmt.Exec(); err != nil {
		return err
	}
	if err = stmt.Close(); err != nil {
		return err
	}

	return txn.Commit()
}

// init connects to postgres
func (s *Surfacer) init() error {
	var err error

	if s.db, err = s.openDB(s.c.GetConnectionString()); err != nil {
		return err
	}
	if err = s.db.Ping(); err != nil {
		return err
	}
	s.writeChan = make(chan *metrics.EventMetrics, 1000)

	// Start a goroutine to run forever, polling on the writeChan. Allows
	// for the surfacer to write asynchronously to the serial port.
	go func() {
		defer s.db.Close()

		for {
			em := <-s.writeChan
			if em.Kind != metrics.CUMULATIVE && em.Kind != metrics.GAUGE {
				continue
			}
			// Note: we may want to batch calls to writeMetrics, as each call results in
			// a database transaction.
			if err := s.writeMetrics(em); err != nil {
				s.l.Warningf("Error while writing metrics: %v", err)
			}
		}
	}()

	return nil
}

// Write takes the data to be written
func (s *Surfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.writeChan <- em:
	default:
		s.l.Warningf("Surfacer's write channel is full, dropping new data.")
	}
}
