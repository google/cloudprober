// Copyright 2020 Google Inc.
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
Package bigquery connects to the GCP BigQuery API and computes metrics suitable
for blackbox-probing.
*/
package bigquery

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
)

// The QueryRunner interface encapsulates the BigQuery API.
//
// Query() runs the given query on BQ and returns the result and an error, if
// any.
type QueryRunner interface {
	Query(context.Context, string) (string, error)
}

type bigqueryRunner struct {
	client *bigquery.Client
}

// Query takes a string in BQL and executes it on BigQuery, returning the
// result.
func (r *bigqueryRunner) Query(ctx context.Context, query string) (string, error) {
	q := r.client.Query(query)

	it, err := q.Read(ctx)
	if err != nil {
		return "", err
	}
	var row []bigquery.Value

	if err = it.Next(&row); err != nil {
		return "", err
	}
	return fmt.Sprint(row[0]), nil
}

// NewRunner creates a new BQ QueryRunner associated with the given projectID.
func NewRunner(ctx context.Context, projectID string) (QueryRunner, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &bigqueryRunner{client}, nil
}

// Probe executes queries against BQ and returns the resulting metrics.
// If no table is specified, Probe will execute a dummy query against BQ (such
// as 'SELECT 1'). Otherwise, table should specify an existing table in the
// format 'dataset.table'.
//
// The service account cloudprober runs under should have sufficient permissions
// to read the table.
func Probe(ctx context.Context, r QueryRunner, table string) (string, error) {

	var metrics, result string
	var err error

	if table == "" {
		result, err = r.Query(ctx, "SELECT 1")
		metrics = fmt.Sprintf("%s %s", "bigquery_connect", result)
	} else {
		result, err = r.Query(ctx, "SELECT COUNT(*) FROM "+table)
		metrics = fmt.Sprintf("%s %s", "row_count", result)
	}

	if err != nil {
		return "", err
	}
	return metrics, nil
}
