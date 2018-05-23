// Copyright 2017 Google Inc.
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

// Package postgres implements "postgres" surfacer. This surfacer type is in
// experimental phase right now.
package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"

	"database/sql"
	"encoding/json"
	configpb "github.com/google/cloudprober/surfacers/postgres/proto"
	"github.com/lib/pq"
	"strconv"
	"time"
)

type label struct {
	key   string
	value string
}

type pgMetric struct {
	time       time.Time
	metricName string
	value      string
	labels     []label
}

// labelsJSON takes the label array and formats it for insertion into
// postgres jsonb labels column, storing each label as k,v json object
func (c pgMetric) labelsJSON() (string, error) {
	m := make(map[string]string)
	for _, l := range c.labels {
		m[l.key] = l.value
	}

	bs, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func newPGMetric(t time.Time, metricName, val string, labels []label) pgMetric {
	return pgMetric{
		time:       t,
		metricName: metricName,
		value:      val,
		labels:     labels,
	}
}

type pgDistribution struct {
	*metrics.DistributionData

	metricName string
	labels     []label
	timestamp  time.Time
}

// sumMetric creates the correctly named metric
// representing sum of the distribution
func (d pgDistribution) sumMetric() pgMetric {
	return newPGMetric(
		d.timestamp,
		d.metricName+"_sum",
		strconv.FormatFloat(d.Sum, 'f', -1, 64),
		d.labels,
	)
}

// countMetric creates the correctly named metric representing
// the count of the distribution
func (d pgDistribution) countMetric() pgMetric {
	return newPGMetric(
		d.timestamp,
		d.metricName+"_count",
		strconv.FormatInt(d.Count, 10),
		d.labels,
	)
}

// bucketMetrics creates and formats all metrics for each bucket in this distribution.
// each bucket is assigned a metric name suffixed with "_bucket" and labeled with the
// corresponding bucket as "le: {bucket}"
func (d pgDistribution) bucketMetrics() []pgMetric {
	var val int64
	ms := []pgMetric{}

	for i := range d.LowerBounds {
		val += d.BucketCounts[i]
		var lb string
		if i == len(d.LowerBounds)-1 {
			lb = "+Inf"
		} else {
			lb = strconv.FormatFloat(d.LowerBounds[i+1], 'f', -1, 64)
		}
		labelsWithBucket := append(d.labels, label{"le", lb})
		ms = append(ms, newPGMetric(d.timestamp, d.metricName+"_bucket", strconv.FormatInt(val, 10), labelsWithBucket))
	}

	return ms
}

// metricRows extracts all metrics to be insterted into postgres
// corresponding to the EventMEtric
func metricRows(em *metrics.EventMetrics) []pgMetric {
	fmt.Printf("%+v\n", em)
	cs := []pgMetric{}

	labels := []label{}

	for _, k := range em.LabelsKeys() {
		labels = append(labels, label{k, em.Label(k)})
	}

	for _, metricName := range em.MetricsKeys() {
		val := em.Metric(metricName)

		if mapVal, ok := val.(*metrics.Map); ok {
			for _, k := range mapVal.Keys() {
				labelsWithMap := append(labels, label{mapVal.MapName, k})
				cs = append(cs, newPGMetric(em.Timestamp, metricName, mapVal.GetKey(k).String(), labelsWithMap))
			}
			continue
		}

		if distVal, ok := val.(*metrics.Distribution); ok {
			d := distVal.Data()
			pgD := pgDistribution{d, metricName, labels, em.Timestamp}

			cs = append(cs,
				pgD.sumMetric(),
				pgD.countMetric(),
			)

			cs = append(cs, pgD.bucketMetrics()...)

			continue
		}

		if _, ok := val.(metrics.String); ok {
			newLabels := append(labels, label{"val", val.String()})
			cs = append(cs, newPGMetric(em.Timestamp, metricName, "1", newLabels))
			continue
		}

		cs = append(cs, newPGMetric(em.Timestamp, metricName, val.String(), labels))
	}
	return cs
}

// PostgresSurfacer structures for writing to postgres.
type PostgresSurfacer struct {
	// Configuration
	c *configpb.SurfacerConf

	// Channel for incoming data.
	writeChan chan *metrics.EventMetrics

	// Cloud logger
	l *logger.Logger

	openDB func(connectionString string) (*sql.DB, error)
	db     *sql.DB
}

// New initializes a PostgresSurfacer for inserting probe results into postgres
func New(config *configpb.SurfacerConf, l *logger.Logger) (*PostgresSurfacer, error) {
	s := &PostgresSurfacer{
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
func (s *PostgresSurfacer) writeMetrics(em *metrics.EventMetrics) error {
	rows := metricRows(em)

	txn, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(
		pq.CopyIn(
			s.c.GetMetricsTableName(), "time", "metric_name", "value", "labels",
		),
	)

	if err != nil {
		return err
	}

	for _, r := range rows {
		s, err := r.labelsJSON()
		if err != nil {
			return err
		}
		_, err = stmt.Exec(r.time, r.metricName, r.value, s)
		if err != nil {
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}

	return nil
}

// init connects to postgres
func (s *PostgresSurfacer) init() error {
	var err error
	fmt.Fprintf(os.Stdout, "%s\n", s.c.GetConnectionString())

	s.db, err = s.openDB(s.c.GetConnectionString())
	if err != nil {
		return err
	}

	err = s.db.Ping()
	if err != nil {
		return err
	}

	s.writeChan = make(chan *metrics.EventMetrics, 1000)

	// Start a goroutine to run forever, polling on the writeChan. Allows
	// for the surfacer to write asynchronously to the serial port.
	go func() {
		defer s.db.Close()

		for {
			em := <-s.writeChan

			// batch all metrics into a sql statement
			if em.Kind != metrics.CUMULATIVE && em.Kind != metrics.GAUGE {
				continue
			}

			err := s.writeMetrics(em)
			if err != nil {
				fmt.Fprintf(os.Stdout, "%+v\n", err)
			}
		}
	}()

	return nil
}

// Write takes the data to be written
func (s *PostgresSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.writeChan <- em:
	default:
		s.l.Warningf("PostgresSurfacer's write channel is full, dropping new data.")
	}
}
