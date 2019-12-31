// Copyright 2019 Google Inc.
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

package http

import (
	"fmt"
	"io"
	"net/http"

	"github.com/google/cloudprober/targets/endpoint"
)

// requestBody encapsulates the request body and implements the io.Reader()
// interface.
type requestBody struct {
	b []byte
}

// Read implements the io.Reader interface. Instead of using buffered read,
// it simply copies the bytes to the provided slice in one go (depending on
// the input slice capacity) and returns io.EOF. Buffered reads require
// resetting the buffer before re-use, restricting our ability to use the
// request object concurrently.
func (rb *requestBody) Read(p []byte) (int, error) {
	return copy(p, rb.b), io.EOF
}

func (p *Probe) httpRequestForTarget(target endpoint.Endpoint) *http.Request {
	// Prepare HTTP.Request for Client.Do
	host := target.Name

	if p.c.GetResolveFirst() {
		ip, err := p.opts.Targets.Resolve(target.Name, p.opts.IPVersion)
		if err != nil {
			p.l.Error("target: ", target.Name, ", resolve error: ", err.Error())
			return nil
		}
		host = ip.String()
	}

	if p.c.GetPort() != 0 {
		host = fmt.Sprintf("%s:%d", host, p.c.GetPort())
	} else if target.Port != 0 {
		host = fmt.Sprintf("%s:%d", host, target.Port)
	}

	url := fmt.Sprintf("%s://%s%s", p.protocol, host, p.url)

	// Prepare request body
	var body io.Reader
	if p.c.GetBody() != "" {
		body = &requestBody{[]byte(p.c.GetBody())}
	}
	req, err := http.NewRequest(p.method, url, body)
	if err != nil {
		p.l.Error("target: ", target.Name, ", error creating HTTP request: ", err.Error())
		return nil
	}

	// If resolving early, URL contains IP for the hostname (see above). Update
	// req.Host after request creation, so that correct Host header is sent to the
	// web server.
	req.Host = target.Name

	for _, header := range p.c.GetHeaders() {
		if header.GetName() == "Host" {
			req.Host = header.GetValue()
			continue
		}
		req.Header.Set(header.GetName(), header.GetValue())
	}

	if p.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.bearerToken)
	}

	return req
}
