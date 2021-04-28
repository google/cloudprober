// Copyright 2018 The Cloudprober Authors.
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

// Package web provides web interface for cloudprober.
package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/google/cloudprober"
	"github.com/google/cloudprober/config/runconfig"
	"github.com/google/cloudprober/probes"
	"github.com/google/cloudprober/servers"
	"github.com/google/cloudprober/surfacers"
	"github.com/google/cloudprober/sysvars"
)

func execTmpl(tmpl *template.Template, v interface{}) template.HTML {
	var statusBuf bytes.Buffer
	err := tmpl.Execute(&statusBuf, v)
	if err != nil {
		return template.HTML(template.HTMLEscapeString(err.Error()))
	}
	return template.HTML(statusBuf.String())
}

// Status returns cloudprober status string.
func Status() string {
	var statusBuf bytes.Buffer

	probeInfo, surfacerInfo, serverInfo := cloudprober.GetInfo()
	startTime := sysvars.StartTime()
	uptime := time.Since(startTime)

	tmpl, _ := template.New("statusTmpl").Parse(statusTmpl)
	tmpl.Execute(&statusBuf, struct {
		Version, StartTime, Uptime, ProbesStatus, ServersStatus, SurfacersStatus interface{}
	}{
		Version:         runconfig.Version(),
		StartTime:       startTime.Format(time.RFC1123),
		Uptime:          uptime.String(),
		ProbesStatus:    execTmpl(probes.StatusTmpl, probeInfo),
		SurfacersStatus: execTmpl(surfacers.StatusTmpl, surfacerInfo),
		ServersStatus:   execTmpl(servers.StatusTmpl, serverInfo),
	})

	return statusBuf.String()
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, cloudprober.GetTextConfig())
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, Status())
}

// Init initializes cloudprober web interface handler.
func Init() {
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/status", statusHandler)
}
