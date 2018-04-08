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
	"fmt"
	"net/http"

	"github.com/google/cloudprober"
)

func configHandler(w http.ResponseWriter, r *http.Request) {
	config := cloudprober.GetConfig()
	fmt.Fprintf(w, config)
}

// Init initializes cloudprober web interface handler.
func Init() {
	http.HandleFunc("/config", configHandler)
	// TODO: Add a status handler that shows the running probes and
	// their configs.
}
