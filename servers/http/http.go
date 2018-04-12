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
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/probeutils"
	"github.com/google/cloudprober/sysvars"
	"github.com/google/cloudprober/targets/lameduck"
)

const statsExportInterval = 10 * time.Second

// OK a string returned as successful indication by "/", and "/healthcheck".
const OK = "ok"

// statsKeeper manages the stats and exports those stats at a regular basis.
// Currently we only maintain the number of requests received per URL.
func (s *Server) statsKeeper(name string, statsChan <-chan string) {
	reqMetric := metrics.NewMap("url", metrics.NewInt(0))
	em := metrics.NewEventMetrics(time.Now()).
		AddMetric("req", reqMetric).
		AddLabel("module", name)
	doExport := time.Tick(s.statsInterval)
	for {
		select {
		case url := <-statsChan:
			reqMetric.IncKey(url)
		case ts := <-doExport:
			em.Timestamp = ts
			s.dataChan <- em.Clone()
			s.l.Info(em.String())
		}
	}
}

// lameduckStatus fetches the global list of lameduck targets and returns:
// - (true, nil) if this machine is in that list
// - (false, nil) if not
// - (false, error) upon failure to fetch the list.
func (s *Server) lameduckStatus() (bool, error) {
	if s.ldLister == nil {
		return false, errors.New("lameduck lister not initialized")
	}

	lameducksList, err := s.ldLister.List()
	if err != nil {
		return false, fmt.Errorf("error getting list of lameducking targets: %v", err)
	}
	for _, lameduckName := range lameducksList {
		if s.instanceName == lameduckName {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) lameduckHandler(w http.ResponseWriter) {
	if lameduck, err := s.lameduckStatus(); err != nil {
		fmt.Fprintf(w, "HTTP Server: Error getting lameduck status: %v", err)
	} else {
		fmt.Fprint(w, lameduck)
	}
}

func (s *Server) healthcheckHandler(w http.ResponseWriter) {
	lameduck, err := s.lameduckStatus()
	if err != nil {
		s.l.Error(err)
	}
	if lameduck {
		http.Error(w, "lameduck", http.StatusServiceUnavailable)
	} else {
		fmt.Fprint(w, OK)
	}
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request, statsChan chan<- string) {
	switch r.URL.Path {
	case "/lameduck":
		s.lameduckHandler(w)
	case "/healthcheck":
		s.healthcheckHandler(w)
	default:
		res, ok := s.staticURLResTable[r.URL.Path]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, res)
	}
	select {
	case statsChan <- r.URL.Path:
	default:
		s.l.Warning("Was not able to send a URL to the stats channel.")
	}
	return
}

// Server implements a basic single-threaded, fast response web server.
type Server struct {
	c                 *ServerConf
	ln                net.Listener
	instanceName      string
	staticURLResTable map[string]string
	dataChan          chan<- *metrics.EventMetrics
	statsInterval     time.Duration
	ldLister          lameduck.Lister // Lameduck lister
	l                 *logger.Logger
}

// New returns a Server.
func New(initCtx context.Context, c *ServerConf, l *logger.Logger) (*Server, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", int(c.GetPort())))
	if err != nil {
		return nil, err
	}

	// If we are not able get the default lameduck lister, we only log a warning.
	ldLister, err := lameduck.GetDefaultLister()
	if err != nil {
		l.Warning(err)
	}

	// Cleanup listener if initCtx is canceled.
	go func() {
		<-initCtx.Done()
		ln.Close()
	}()

	return &Server{
		c:             c,
		l:             l,
		ln:            ln,
		ldLister:      ldLister,
		statsInterval: statsExportInterval,
		instanceName:  sysvars.Vars()["instance"],
		staticURLResTable: map[string]string{
			"/":         OK,
			"/instance": sysvars.Vars()["instance"],
		},
	}, nil
}

// Start starts a simple HTTP server on a given port. This function returns
// only if there is an error.
func (s *Server) Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error {
	s.dataChan = dataChan

	// 100 outstanding stats update requests
	statsChan := make(chan string, 100)

	laddr := s.ln.Addr().String()
	go s.statsKeeper(fmt.Sprintf("http-server-%s", laddr), statsChan)

	for _, dh := range s.c.GetPatternDataHandler() {
		size := int(dh.GetResponseSize())
		resp := string(probeutils.PatternPayload([]byte(dh.GetPattern()), size))
		s.staticURLResTable[fmt.Sprintf("/data_%d", size)] = resp
	}

	// Not using default server mux as we may run multiple HTTP servers, e.g. for testing.
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.handler(w, r, statsChan)
	})
	s.l.Infof("Starting HTTP server at: %s", laddr)
	srv := &http.Server{
		Addr:         laddr,
		Handler:      serverMux,
		ReadTimeout:  time.Duration(s.c.GetReadTimeoutMs()) * time.Millisecond,
		WriteTimeout: time.Duration(s.c.GetWriteTimeoutMs()) * time.Millisecond,
		IdleTimeout:  time.Duration(s.c.GetIdleTimeoutMs()) * time.Millisecond,
	}

	// Setup a background function to close server if context is canceled.
	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	// Following returns only in case of an error.
	return srv.Serve(s.ln)
}
