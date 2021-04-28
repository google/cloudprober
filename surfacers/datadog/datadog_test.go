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

package datadog

import (
	"context"
  "testing"
	"time"
  "reflect"

	"github.com/google/cloudprober/logger"
	"github.com/google/cloudprober/metrics"
)

func newTestDDSurfacer() DDSurfacer {
  l, _ := logger.New(context.TODO(), "test-logger")

  return DDSurfacer{
    l: l,
    prefix: "testPrefix",
  }
}

func TestEmLabelsToTags(t *testing.T) {
  timestamp := time.Now()

  tests := map[string]struct {
      em *metrics.EventMetrics
      want []string
  }{
    "no label": {
        em: metrics.NewEventMetrics(timestamp),
        want: []string{},
    },
    "one label": {
        em: metrics.NewEventMetrics(timestamp).AddLabel("ptype", "sysvars"),
        want: []string{"ptype:sysvars"},
    },
    "three labels": {
        em: metrics.NewEventMetrics(timestamp).AddLabel("label1", "value1").
                                               AddLabel("label2", "value2").
                                               AddLabel("label3", "value3"),
        want: []string{"label1:value1","label2:value2","label3:value3"},
    },
  }

  for name, tc := range tests {
    t.Run(name, func(t *testing.T) {
      got := emLabelsToTags(tc.em)
      if !reflect.DeepEqual(got, tc.want) {
     // if got != tc.want {
        t.Errorf("got: %v, want %v %v %v", got, tc.want, reflect.TypeOf(got), reflect.TypeOf(tc.want))
      }
    })
  }
}

