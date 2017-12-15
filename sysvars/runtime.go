package sysvars

import (
	"runtime"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"google3/third_party/golang/go_sys/unix/unix"
)

func runtimeVars(dataChan chan *metrics.EventMetrics, l *logger.Logger) {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	ts := time.Now()
	counterRuntimeVars(dataChan, ts, m, l)
	gaugeRuntimeVars(dataChan, ts, m, l)
}

// counterRuntimeVars exports counter runtime stats, stats that grow through
// the lifetime of the process. These stats are exported as CUMULATIVE
// EventMetrics.
func counterRuntimeVars(dataChan chan *metrics.EventMetrics, ts time.Time, m *runtime.MemStats, l *logger.Logger) {
	em := metrics.NewEventMetrics(ts).
		AddLabel("ptype", "sysvars").
		AddLabel("probe", "sysvars")

	// Time since this module started.
	timeSince := time.Since(startTime).Seconds()
	em.AddMetric("uptime_msec", metrics.NewFloat(timeSince*1000))
	// TODO: Deprecate "uptime" in favor of "uptime_msec".
	em.AddMetric("uptime", metrics.NewInt(int64(timeSince)))

	// GC memory stats
	em.AddMetric("gc_time_msec", metrics.NewFloat(float64(m.PauseTotalNs)/1e6))
	em.AddMetric("mallocs", metrics.NewInt(int64(m.Mallocs)))
	em.AddMetric("frees", metrics.NewInt(int64(m.Frees)))

	// CPU usage
	var timeSpec unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_PROCESS_CPUTIME_ID, &timeSpec); err != nil {
		l.Warningf("Error while trying to get CPU usage: %v", err)
	} else {
		em.AddMetric("cpu_usage_msec", metrics.NewFloat(float64(timeSpec.Nano())/1e6))
	}

	dataChan <- em
	l.Info(em.String())
}

// gaugeRuntimeVars exports gauge runtime stats, stats that represent the
// current state and may go up or down. These stats are exported as GAUGE
// EventMetrics.
func gaugeRuntimeVars(dataChan chan *metrics.EventMetrics, ts time.Time, m *runtime.MemStats, l *logger.Logger) {
	em := metrics.NewEventMetrics(ts).
		AddLabel("ptype", "sysvars").
		AddLabel("probe", "sysvars")
	em.Kind = metrics.GAUGE

	// Number of goroutines
	em.AddMetric("goroutines", metrics.NewInt(int64(runtime.NumGoroutine())))
	// Overall memory being used by the Go runtime (in bytes).
	em.AddMetric("mem_stats_sys_bytes", metrics.NewInt(int64(m.Sys)))

	dataChan <- em
	l.Info(em.String())
}
