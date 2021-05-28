// Copyright 2017 The Cloudprober Authors.
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
	"html/template"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
	"github.com/google/cloudprober/servers/external"
	"github.com/google/cloudprober/servers/grpc"
	"github.com/google/cloudprober/servers/http"
	configpb "github.com/google/cloudprober/servers/proto"
	"github.com/google/cloudprober/servers/udp"
	"github.com/google/cloudprober/web/formatutils"
)

// StatusTmpl variable stores the HTML template suitable to generate the
// servers' status for cloudprober's /status page. It expects an array of
// ServerInfo objects as input.
var StatusTmpl = template.Must(template.New("statusTmpl").Parse(`
<table class="status-list">
  <tr>
    <th>Type</th>
    <th>Conf</th>
  </tr>
  {{ range . }}
  <tr>
    <td>{{.Type}}</td>
    <td>
    {{if .Conf}}
      <pre>{{.Conf}}</pre>
    {{else}}
      default
    {{end}}
    </td>
  </tr>
  {{ end }}
</table>
`))

// Server interface has only one method: Start.
type Server interface {
	Start(ctx context.Context, dataChan chan<- *metrics.EventMetrics) error
}

// ServerInfo encapsulates a Server and related info.
type ServerInfo struct {
	Server
	Type string
	Conf string
}

// Init initializes cloudprober servers, based on the provided config.
func Init(initCtx context.Context, serverDefs []*configpb.ServerDef) (servers []*ServerInfo, err error) {
	for _, serverDef := range serverDefs {
		var l *logger.Logger
		l, err = logger.NewCloudproberLog(serverDef.GetType().String())
		if err != nil {
			return
		}

		var conf interface{}
		var server Server

		switch serverDef.GetType() {
		case configpb.ServerDef_HTTP:
			server, err = http.New(initCtx, serverDef.GetHttpServer(), l)
			conf = serverDef.GetHttpServer()
		case configpb.ServerDef_UDP:
			server, err = udp.New(initCtx, serverDef.GetUdpServer(), l)
			conf = serverDef.GetUdpServer()
		case configpb.ServerDef_GRPC:
			server, err = grpc.New(initCtx, serverDef.GetGrpcServer(), l)
			conf = serverDef.GetGrpcServer()
		case configpb.ServerDef_EXTERNAL:
			server, err = external.New(initCtx, serverDef.GetExternalServer(), l)
			conf = serverDef.GetExternalServer()
		}
		if err != nil {
			return
		}

		servers = append(servers, &ServerInfo{
			Server: server,
			Type:   serverDef.GetType().String(),
			Conf:   formatutils.ConfToString(conf),
		})
	}
	return
}
