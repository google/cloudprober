// Copyright 2017 Google Inc.
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
servers package provides an interface to initialize cloudprober servers using servers config.
*/
package servers

import (
	"context"

	"github.com/golang/glog"
	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/servers/http"
	"github.com/google/cloudprober/servers/udp"
)

const (
	logsNamePrefix = "cloudprober"
)

func newLogger(ctx context.Context, logName string) (*logger.Logger, error) {
	return logger.New(ctx, logsNamePrefix+"."+logName)
}

// Start initializes and starts cloudprober servers, based on the provided config.
func Start(ctx context.Context, serverProtobufs []*Server, dataChan chan<- *metrics.EventMetrics) {
	for _, s := range serverProtobufs {
		runServer(ctx, s, dataChan)
	}
}

func runServer(ctx context.Context, s *Server, dataChan chan<- *metrics.EventMetrics) {
	l, err := newLogger(ctx, s.GetType().String())
	if err != nil {
		glog.Exitf("Error initializing logger for %s. Err: %v", s.GetType().String(), err)
	}
	switch s.GetType() {
	case Server_HTTP:
		go func() {
			glog.Exit(http.ListenAndServe(ctx, s.GetHttpServer(), dataChan, l))
		}()
	case Server_UDP:
		go func() {
			glog.Exit(udp.ListenAndServe(ctx, s.GetUdpServer(), l))
		}()
	}
}
