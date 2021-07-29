// Copyright 2017-2021 The Cloudprober Authors.
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

//go:build linux
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
	l.Debug(em.String())
}
