// Copyright 2018 Google Inc.
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

	"github.com/google/cloudprober"
)

const style = `
<style type="text/css">
table.status-list {
  border-collapse: collapse;
  border-spacing: 0;
	margin-bottom: 40px;
	font-family: monospace;
}
table.status-list td,th {
  border: 1px solid gray;
  padding: 0.25em 0.5em;
	max-width: 200px;
}
pre {
    white-space: pre-wrap;
		word-wrap: break-word;
}
</style>`

const probeStatusTmpl = `

<h3>Probes:</h3>

<table class="status-list">
  <tr>
    <th>Name</th>
    <th>Type</th>
    <th>Interval</th>
    <th>Timeout</th>
    <th>Targets</th>
    <th>Probe Conf</th>
    <th>Latency Unit</th>
    <th>Latency Distribution Lower Bounds (if configured) </th>
  </tr>
  {{ range . }}
  <tr>
    <td>{{.Name}}</td>
    <td>{{.Type}}</td>
    <td>{{.Interval}}</td>
    <td>{{.Timeout}}</td>
    <td><pre>{{.TargetsDesc}}</pre></td>

    <td>
    {{if .ProbeConf}}
      <pre>{{.ProbeConf}}</pre>
    {{else}}
      default
    {{end}}
    </td>

    <td>{{.LatencyUnit}}</td>
    <td><pre>{{.LatencyDistLB}}</pre></td>
  </tr>
  {{ end }}
</table>`

const surfacerStatusTmpl = `
<table class="status-list">
  <tr>
    <th>Type</th>
    <th>Name</th>
    <th>Conf</th>
  </tr>
  {{ range . }}
  <tr>
    <td>{{.Type}}</td>
    <td>{{.Name}}</td>
    <td>
    {{if .Conf}}
      <pre>{{.Conf}}</pre>
    {{else}}
      default
    {{end}}
    </td>
  </tr>
  {{ end }}
</table>`

// Status returns cloudprober status string.
func Status() string {
	var statusBuf bytes.Buffer

	statusBuf.WriteString(style)
	probeInfo, surfacerInfo, serverInfo := cloudprober.GetInfo()

	// Probes info
	tmpl, _ := template.New("statusProber").Parse(probeStatusTmpl)
	tmpl.Execute(&statusBuf, probeInfo)

	// Surfacers info
	statusBuf.WriteString("<h3>Surfacers</h3>")
	tmpl, _ = template.New("statusSurfacer").Parse(surfacerStatusTmpl)
	tmpl.Execute(&statusBuf, surfacerInfo)

	// Servers info
	statusBuf.WriteString("<h3>Servers</h3>")
	tmpl, _ = template.New("statusServer").Parse(surfacerStatusTmpl)
	tmpl.Execute(&statusBuf, serverInfo)

	return statusBuf.String()
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, cloudprober.GetConfig())
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, Status())
}

// Init initializes cloudprober web interface handler.
func Init() {
	http.HandleFunc("/config", configHandler)
	http.HandleFunc("/status", statusHandler)
}
