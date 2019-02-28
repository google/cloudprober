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
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// requestBody encapsulates the request body and implements the io.ReadCloser()
// interface.
type requestBody struct {
	b *bytes.Reader
}

func (rb *requestBody) Read(p []byte) (int, error) {
	return rb.b.Read(p)
}

// Close resets the internal buffer's next read location, to make it ready for
// the next HTTP request.
func (rb *requestBody) Close() error {
	rb.b.Seek(0, io.SeekStart)
	return nil
}

func (p *Probe) httpRequestForTarget(target string) *http.Request {
	// Prepare HTTP.Request for Client.Do
	host := target

	if p.c.GetResolveFirst() {
		ip, err := p.opts.Targets.Resolve(target, 4) // Support IPv4 for now, should be a config option.
		if err != nil {
			p.l.Error("target: ", target, ", resolve error: ", err.Error())
			return nil
		}
		host = ip.String()
	}

	if p.c.GetPort() != 0 {
		host = fmt.Sprintf("%s:%d", host, p.c.GetPort())
	}

	url := fmt.Sprintf("%s://%s%s", p.protocol, host, p.url)

	// Prepare request body
	body := &requestBody{
		b: bytes.NewReader([]byte(p.c.GetBody())),
	}
	req, err := http.NewRequest(p.method, url, body)
	if err != nil {
		p.l.Error("target: ", target, ", error creating HTTP request: ", err.Error())
		return nil
	}

	// If resolving early, URL contains IP for the hostname (see above). Update
	// req.Host after request creation, so that correct Host header is sent to the
	// web server.
	req.Host = target

	for _, header := range p.c.GetHeaders() {
		req.Header.Set(header.GetName(), header.GetValue())
	}

	return req
}
