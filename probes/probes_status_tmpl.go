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

package probes

import "html/template"

// StatusTmpl variable stores the HTML template suitable to generate the
// probes' status for cloudprober's /status page. It expects an array of
// ProbeInfo objects as input.
var StatusTmpl = template.Must(template.New("statusTmpl").Parse(`
<table class="status-list">
  <tr>
    <th>Name</th>
    <th>Type</th>
    <th>Interval</th>
    <th>Timeout</th>
    <th width="20%%">Targets</th>
    <th width="30%%">Probe Conf</th>
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
</table>
`))
