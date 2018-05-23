package sysvars

import (
	"runtime"
	"testing"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

func TestCounterRuntimeVars(t *testing.T) {
	dataChan := make(chan *metrics.EventMetrics, 1)
	l := &logger.Logger{}
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	ts := time.Now()

	counterRuntimeVars(dataChan, ts, m, l)
	em := <-dataChan

	if em.Timestamp != ts {
		t.Errorf("em.Timestamp=%v, want=%v", em.Timestamp, ts)
	}

	if em.Kind != metrics.CUMULATIVE {
		t.Errorf("Metrics kind is not cumulative.")
	}

	for _, name := range []string{"uptime_msec", "gc_time_msec", "mallocs", "frees"} {
		if em.Metric(name) == nil {
			t.Errorf("Expected metric \"%s\" not defined in EventMetrics: %s", name, em.String())
		}
	}
}

func TestGaugeRuntimeVars(t *testing.T) {
	dataChan := make(chan *metrics.EventMetrics, 1)
	l := &logger.Logger{}
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	ts := time.Now()

	gaugeRuntimeVars(dataChan, ts, m, l)
	em := <-dataChan

	if em.Timestamp != ts {
		t.Errorf("em.Timestamp=%v, want=%v", em.Timestamp, ts)
	}

	if em.Kind != metrics.GAUGE {
		t.Errorf("Metrics kind is not gauge.")
	}

	for _, name := range []string{"goroutines", "mem_stats_sys_bytes"} {
		if em.Metric(name) == nil {
			t.Errorf("Expected metric \"%s\" not defined in EventMetrics: %s", name, em.String())
		}
	}
}
