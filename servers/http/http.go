// Copyright 2017-2019 The Cloudprober Authors.
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
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/probes/probeutils"
	configpb "github.com/google/cloudprober/servers/http/proto"
	"github.com/google/cloudprober/sysvars"
	"github.com/google/cloudprober/targets/endpoint"
	"github.com/google/cloudprober/targets/lameduck"
)

const statsExportInterval = 10 * time.Second

// OK is the response returned as successful indication by "/", and "/healthcheck".
var OK = "ok"

// statsKeeper manages the stats and exports those stats at a regular basis.
// Currently we only maintain the number of requests received per URL.
func (s *Server) statsKeeper(name string) {
	doExport := time.Tick(s.statsInterval)
	for {
		select {
		case ts := <-doExport:
			em := metrics.NewEventMetrics(ts).
				AddMetric("req", s.reqMetric).
				AddLabel("module", name)
			s.dataChan <- em
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

	lameducksList := s.ldLister.ListEndpoints()
	for _, ep := range lameducksList {
		if s.instanceName == ep.Name {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) lameduckHandler(w http.ResponseWriter) {
	if lameduck, err := s.lameduckStatus(); err != nil {
		fmt.Fprintf(w, "HTTP Server: Error getting lameduck status: %v", err)
	} else {
		w.Write([]byte(strconv.FormatBool(lameduck)))
	}
}

func (s *Server) healthcheckHandler(w http.ResponseWriter) {
	lameduck, err := s.lameduckStatus()
	if err != nil {
		s.l.Error(err.Error())
	}
	if lameduck {
		http.Error(w, "lameduck", http.StatusServiceUnavailable)
	} else {
		w.Write([]byte(OK))
	}
}

func (s *Server) metadataHandler(w http.ResponseWriter, r *http.Request) {
	varNames, ok := r.URL.Query()["var"]
	if !ok || len(varNames) == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	val, ok := s.sysVars[varNames[0]]
	if !ok {
		http.Error(w, fmt.Sprintf("'%s' not found", varNames[0]), http.StatusNotFound)
		return
	}
	w.Write([]byte(val))
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/lameduck":
		s.lameduckHandler(w)
	case "/healthcheck":
		s.healthcheckHandler(w)
	case "/metadata":
		s.metadataHandler(w, r)
	default:
		res, ok := s.staticURLResTable[r.URL.Path]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write(res)
	}
	s.reqMetric.IncKey(r.URL.Path)
}

// Server implements a basic single-threaded, fast response web server.
type Server struct {
	c                 *configpb.ServerConf
	ln                net.Listener
	instanceName      string
	sysVars           map[string]string
	staticURLResTable map[string][]byte
	reqMetric         *metrics.Map
	dataChan          chan<- *metrics.EventMetrics
	statsInterval     time.Duration
	ldLister          endpoint.Lister // Lameduck lister
	l                 *logger.Logger
}

// New returns a Server.
func New(initCtx context.Context, c *configpb.ServerConf, l *logger.Logger) (*Server, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", int(c.GetPort())))
	if err != nil {
		return nil, err
	}

	// If we are not able get the default lameduck lister, we only log a warning.
	ldLister, err := lameduck.GetDefaultLister()
	if err != nil {
		l.Warning(err.Error())
	}

	if c.GetProtocol() == configpb.ServerConf_HTTPS {
		if c.GetTlsCertFile() == "" || c.GetTlsKeyFile() == "" {
			return nil, errors.New("tls_cert_file and tls_key_file are required for HTTPS servers")
		}
	}

	// Cleanup listener if initCtx is canceled.
	go func() {
		<-initCtx.Done()
		ln.Close()
	}()

	sysVars := sysvars.Vars()

	return &Server{
		c:             c,
		l:             l,
		ln:            ln,
		ldLister:      ldLister,
		sysVars:       sysVars,
		reqMetric:     metrics.NewMap("url", metrics.NewInt(0)),
		statsInterval: statsExportInterval,
		instanceName:  sysvars.Vars()["instance"],
		staticURLResTable: map[string][]byte{
			"/":         []byte(OK),
			"/instance": []byte(sysVars["instance"]),
		},
	}, nil
}

// Start starts a simple HTTP server on a given port. This function returns
// only if there is an error.
func (s *Server) Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error {
	s.dataChan = dataChan

	laddr := s.ln.Addr().String()
	go s.statsKeeper(fmt.Sprintf("http-server-%s", laddr))

	for _, dh := range s.c.GetPatternDataHandler() {
		payload := make([]byte, int(dh.GetResponseSize()))
		probeutils.PatternPayload(payload, []byte(dh.GetPattern()))
		s.staticURLResTable[fmt.Sprintf("/data_%d", dh.GetResponseSize())] = payload
	}

	// Not using default server mux as we may run multiple HTTP servers, e.g. for testing.
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", s.handler)
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

	// HTTP/2 is enabled by default for HTTPS servers. To disable it, TLSNextProto
	// should be non-nil and set to an empty dict.
	if s.c.GetDisableHttp2() {
		srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}

	// Following returns only in case of an error.
	if s.c.GetProtocol() == configpb.ServerConf_HTTP {
		return srv.Serve(s.ln)
	}
	return srv.ServeTLS(s.ln, s.c.GetTlsCertFile(), s.c.GetTlsKeyFile())
}
