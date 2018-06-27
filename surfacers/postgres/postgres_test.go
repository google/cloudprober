package postgres

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/cloudprober/metrics"
)

func Test_emToPGMetrics_No_Distribution(t *testing.T) {
	respCodesVal := metrics.NewMap("code", metrics.NewInt(0))
	respCodesVal.IncKeyBy("200", metrics.NewInt(19))
	ts := time.Now()
	em := metrics.NewEventMetrics(ts).
		AddMetric("sent", metrics.NewInt(32)).
		AddMetric("rcvd", metrics.NewInt(22)).
		AddMetric("latency", metrics.NewFloat(10.11111)).
		AddMetric("resp_code", respCodesVal).
		AddLabel("ptype", "http")

	rows := emToPGMetrics(em)

	if len(rows) != 4 {
		t.Errorf("Expected %d rows, received: %d\n", 4, len(rows))
	}

	if !isRowExpected(rows[0], ts, "sent", "32", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[0])
	}

	if !isRowExpected(rows[1], ts, "rcvd", "22", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[1])
	}

	if !isRowExpected(rows[2], ts, "latency", "10.111", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[2])
	}

	if !isRowExpected(rows[3], ts, "resp_code", "19", map[string]string{"ptype": "http", "code": "200"}) {
		t.Errorf("Incorrect Row found %+v", rows[3])
	}
}

func Test_emToPGMetrics_With_Distribution(t *testing.T) {
	respCodesVal := metrics.NewMap("code", metrics.NewInt(0))
	respCodesVal.IncKeyBy("200", metrics.NewInt(19))
	latencyVal := metrics.NewDistribution([]float64{1, 4})
	latencyVal.AddSample(0.5)
	latencyVal.AddSample(5)
	ts := time.Now()
	em := metrics.NewEventMetrics(ts).
		AddMetric("latency", latencyVal).
		AddLabel("ptype", "http")

	rows := emToPGMetrics(em)

	if len(rows) != 5 {
		t.Errorf("Expected %d rows, received: %d\n", 5, len(rows))
	}

	if !isRowExpected(rows[0], ts, "latency_sum", "5.5", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[0])
	}

	if !isRowExpected(rows[1], ts, "latency_count", "2", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[1])
	}

	if !isRowExpected(rows[2], ts, "latency_bucket", "1", map[string]string{"ptype": "http", "le": "1"}) {
		t.Errorf("Incorrect Row found %+v", rows[2])
	}

	if !isRowExpected(rows[3], ts, "latency_bucket", "1", map[string]string{"ptype": "http", "le": "4"}) {
		t.Errorf("Incorrect Row found %+v", rows[3])
	}

	if !isRowExpected(rows[4], ts, "latency_bucket", "2", map[string]string{"ptype": "http", "le": "+Inf"}) {
		t.Errorf("Incorrect Row found %+v", rows[4])
	}

}

func isRowExpected(row pgMetric, t time.Time, metricName string, value string, labels map[string]string) bool {
	if row.time != t {
		return false
	}
	if row.metricName != metricName {
		return false
	}
	if row.value != value {
		return false
	}
	if !reflect.DeepEqual(row.labels, labels) {
		return false
	}

	return true
}
