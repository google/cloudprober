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
Package servers provides an interface to initialize cloudprober servers using servers config.
*/
package servers

import (
	"context"

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

// Server interface has only one method: Start.
type Server interface {
	Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error
}

// Init initializes cloudprober servers, based on the provided config.
func Init(initCtx context.Context, serverDefs []*ServerDef) (servers []Server, err error) {
	for _, serverDef := range serverDefs {
		var l *logger.Logger
		l, err = newLogger(initCtx, serverDef.GetType().String())
		if err != nil {
			return
		}
		var server Server
		switch serverDef.GetType() {
		case ServerDef_HTTP:
			server, err = http.New(initCtx, serverDef.GetHttpServer(), l)
		case ServerDef_UDP:
			server, err = udp.New(initCtx, serverDef.GetUdpServer(), l)
		}
		if err != nil {
			return
		}
		servers = append(servers, server)
	}
	return
}
