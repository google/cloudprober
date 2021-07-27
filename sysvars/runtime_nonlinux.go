//go:build !linux
// +build !linux

package sysvars

import (
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

// osRuntimeVars doesn't anything for the non-Linux systems yet. We have it
// here to make sysvars package compilation work on non-Linux systems.
func osRuntimeVars(dataChan chan *metrics.EventMetrics, l *logger.Logger) {
}
