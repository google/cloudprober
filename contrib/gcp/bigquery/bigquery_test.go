package bigquery

import (
	"context"
	"errors"
	"testing"
)

type stubRunner struct {
	result, query string
	err           error
}

func (f *stubRunner) Query(ctx context.Context, query string) (string, error) {
	f.query = query
	return f.result, f.err
}

func TestProbe(t *testing.T) {
	probeTests := []struct {
		table, expQuery string
		result          string
		err             error
		expMetrics      string
	}{
		{"", "SELECT 1", "1", nil, "bigquery_connect 1"},
		{"ds.table", "SELECT COUNT(*) FROM ds.table", "500", nil, "row_count 500"},
		{"", "SELECT 1", "", errors.New("connection error"), ""},
	}

	for _, pt := range probeTests {
		f := stubRunner{
			result: pt.result,
			err:    pt.err,
		}
		metrics, err := Probe(context.Background(), &f, pt.table)
		if err != pt.err {
			t.Errorf("Probe(table=%#v): mismatched error, got %#v want %#v", pt.table, err, pt.err)
		}
		if metrics != pt.expMetrics {
			t.Errorf("Probe(table=%#v) = %#v, want %#v", pt.table, metrics, pt.expMetrics)
		}
		if f.query != pt.expQuery {
			t.Errorf("Probe(table=%#v): unexpected BQL, got: %#v, want %#v", pt.table, f.query, pt.expQuery)
		}
	}
}
