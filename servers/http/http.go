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
	"github.com/google/cloudprober/targets/lameduck"
)

const statsExportInterval = 10 * time.Second

// OK a string returned as successful indication by "/", and "/healthcheck".
const OK = "ok"

var (
	lameduckGetDefaultLister = lameduck.GetDefaultLister
)

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

// lameduckStatus fetches the global list of lameduck targets and returns:
// - (true, nil) if this machine is in that list
// - (false, nil) if not
// - (false, error) upon failure to fetch the list.
func lameduckStatus(instanceName string) (bool, error) {
	lameduckLister, err := lameduckGetDefaultLister()
	if err != nil {
		return false, fmt.Errorf("getting lameduck lister service: %v", err)
	}

	lameducksList, err := lameduckLister.List()
	if err != nil {
		return false, fmt.Errorf("getting list of lameducking targets: %v", err)
	}
	for _, lameduckName := range lameducksList {
		if instanceName == lameduckName {
			return true, nil
		}
	}
	return false, nil
}

func lameduckHandler(w http.ResponseWriter, instanceName string, l *logger.Logger) {
	if lameduck, err := lameduckStatus(instanceName); err != nil {
		fmt.Fprintf(w, "HTTP Server: Error getting lameduck status: %v", err)
	} else {
		fmt.Fprint(w, lameduck)
	}
}

func healthcheckHandler(w http.ResponseWriter, instanceName string, l *logger.Logger) {
	lameduck, err := lameduckStatus(instanceName)
	if err != nil {
		l.Error(err)
	}
	if lameduck {
		http.Error(w, "lameduck", http.StatusServiceUnavailable)
	} else {
		fmt.Fprint(w, OK)
	}
}

func handler(w http.ResponseWriter, r *http.Request, instanceName string, statsChan chan<- string, l *logger.Logger) {
	switch r.URL.Path {
	case "/":
		fmt.Fprint(w, OK)
	case "/instance":
		fmt.Fprint(w, instanceName)
	case "/lameduck":
		lameduckHandler(w, instanceName, l)
	case "/healthcheck":
		healthcheckHandler(w, instanceName, l)
	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	select {
	case statsChan <- r.URL.Path:
	default:
		l.Warning("was not able to send a URL to the stats channel")
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

	laddr := ln.Addr().String()
	go statsKeeper(fmt.Sprintf("http-server-%s", laddr), statsChan, dataChan, statsExportInterval, l)

	// Not using default server mux as we may run multiple HTTP servers, e.g. for testing.
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, sysVars["instance"], statsChan, l)
	})
	l.Infof("starting HTTP server at: %s", laddr)
	srv := &http.Server{Addr: laddr, Handler: serverMux}

	// Setup a background function to close server if context is canceled.
	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	// Following returns only in case of an error.
	return srv.Serve(ln)
}
