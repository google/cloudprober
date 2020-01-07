// Copyright 2020 Google Inc.
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
Bigquery_probe is an external Cloudprober probe suitable for blackbox-probing
the GCP BigQuery API.

This binary assumes that the environment variable
$GOOGLE_APPLICATION_CREDENTIALS has been set following the instructions at
https://cloud.google.com/docs/authentication/production
*/
package main

import (
	"context"
	"fmt"

	"flag"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	serverpb "github.com/google/cloudprober/probes/external/proto"
	"github.com/google/cloudprober/probes/external/serverutils"
	"google3/third_party/cloudprober/contrib/gcp/bigquery"
)

var (
	projectID  = flag.String("project_id", "", "GCP project ID to connect to.")
	serverMode = flag.Bool("server_mode", false, "Whether to run in server mode.")
	table      = flag.String("table", "", "Table to probe (specified as \"dataset.table\"). "+
		"If empty, the probe will simply try to connect to BQ and execute 'SELECT 1'. "+
		"If running in server mode, the 'table' option from the configuration will override this flag.")
)

func parseProbeRequest(request *serverpb.ProbeRequest) map[string]string {
	m := make(map[string]string)
	for _, opt := range request.Options {
		m[*opt.Name] = *opt.Value
	}
	return m
}

func main() {
	flag.Parse()

	if *projectID == "" {
		glog.Exitf("--project_id must be specified")
	}

	dstTable := *table
	ctx := context.Background()
	runner, err := bigquery.NewRunner(ctx, *projectID)
	if err != nil {
		glog.Fatal(err)
	}

	if *serverMode {
		serverutils.Serve(func(request *serverpb.ProbeRequest, reply *serverpb.ProbeReply) {

			opts := parseProbeRequest(request)
			if val, ok := opts["table"]; ok {
				dstTable = val
				glog.Infof("--table set to %q by ProbeRequest config", val)
			}
			payload, err := bigquery.Probe(ctx, runner, dstTable)
			reply.Payload = proto.String(payload)
			if err != nil {
				reply.ErrorMessage = proto.String(err.Error())
			}
		})
	}

	payload, err := bigquery.Probe(ctx, runner, dstTable)
	if err != nil {
		glog.Fatal(err)
	}
	fmt.Println(payload)

}
