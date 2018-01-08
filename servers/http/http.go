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

// Package http implements an HTTP server that simply returns 'ok' for any URL and sends stats
// on a string channel. This is used by cloudprober to act as the backend for the HTTP based
// probes.
package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/sysvars"
)

const statsExportInterval = 10 * time.Second

var defaultResponse = "ok"

// statsKeeper manages the stats and exports those stats at a regular basis.
// Currently we only maintain the number of requests received per URL.
func statsKeeper(name string, statsChan <-chan string, dataChan chan<- *metrics.EventMetrics, exportInterval time.Duration, l *logger.Logger) {
	reqMetric := metrics.NewMap("url", metrics.NewInt(0))
	em := metrics.NewEventMetrics(time.Now()).
		AddMetric("req", reqMetric).
		AddLabel("module", name)
	doExport := time.Tick(exportInterval)
	for {
		select {
		case url := <-statsChan:
			reqMetric.IncKey(url)
		case ts := <-doExport:
			em.Timestamp = ts
			dataChan <- em.Clone()
			l.Info(em.String())
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request, urlResTable map[string]string, statsChan chan<- string, l *logger.Logger) {
	res, ok := urlResTable[r.URL.Path]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	fmt.Fprint(w, res)
	select {
	case statsChan <- r.URL.Path:
	default:
		l.Warning("Was not able to send a URL to the stats channel.")
	}
	return
}

// ListenAndServe starts a simple HTTP server on a given port. This function never returns
// unless there is an error.
func ListenAndServe(ctx context.Context, c *ServerConf, dataChan chan<- *metrics.EventMetrics, l *logger.Logger) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", int(c.GetPort())))
	if err != nil {
		return err
	}
	return serve(ctx, ln, dataChan, sysvars.Vars(), statsExportInterval, l)
}

func serve(ctx context.Context, ln net.Listener, dataChan chan<- *metrics.EventMetrics, sysVars map[string]string, statsExportInterval time.Duration, l *logger.Logger) error {
	// 1000 outstanding stats update requests
	statsChan := make(chan string, 1000)
	urlResTable := map[string]string{
		"/":         defaultResponse,
		"/instance": sysVars["instance"],
	}

	laddr := ln.Addr().String()
	go statsKeeper(fmt.Sprintf("http-server-%s", laddr), statsChan, dataChan, statsExportInterval, l)

	// Not using default server mux as we may run multiple HTTP servers, e.g. for testing.
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, urlResTable, statsChan, l)
	})
	l.Infof("Starting HTTP server at: %s", laddr)
	srv := &http.Server{Addr: laddr, Handler: serverMux}

	// Setup a background function to close server if context is canceled.
	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	// Following returns only in case of an error.
	return srv.Serve(ln)
}
