//+build linux

package sysvars

import (
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"golang.org/x/sys/unix"
)

func osRuntimeVars(dataChan chan *metrics.EventMetrics, l *logger.Logger) {
	em := metrics.NewEventMetrics(time.Now()).
		AddLabel("ptype", "sysvars").
		AddLabel("probe", "sysvars")

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
