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
	"net"
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

// resolveFunc resolves the given host for the IP version.
// This type is mainly used for testing. For all other cases, a nil function
// should be passed to the httpRequestForTarget function.
type resolveFunc func(host string, ipVer int) (net.IP, error)

func (p *Probe) httpRequestForTarget(target endpoint.Endpoint, resolveF resolveFunc) *http.Request {
	// Prepare HTTP.Request for Client.Do
	port := int(p.c.GetPort())
	// If port is not configured explicitly, use target's port if available.
	if port == 0 {
		port = target.Port
	}

	urlHost, hostHeader := target.Name, target.Name

	if p.c.GetResolveFirst() {
		if resolveF == nil {
			resolveF = p.opts.Targets.Resolve
		}

		ip, err := resolveF(target.Name, p.opts.IPVersion)
		if err != nil {
			p.l.Error("target: ", target.Name, ", resolve error: ", err.Error())
			return nil
		}
		urlHost = ip.String()
	}

	if port != 0 {
		urlHost = fmt.Sprintf("%s:%d", urlHost, port)
		hostHeader = fmt.Sprintf("%s:%d", hostHeader, port)
	}

	url := fmt.Sprintf("%s://%s%s", p.protocol, urlHost, p.url)

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
	// req.Host after request creation, so that correct Host header is sent to
	// the web server.
	req.Host = hostHeader

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
