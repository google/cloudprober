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

package kubernetes

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/cloudprober/logger"
)

// Variables defined by Kubernetes spec to find out local CA cert and token.
const (
	LocalCACert    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	LocalTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// client encapsulates an in-cluster kubeapi client.
type client struct {
	httpC   *http.Client
	apiHost string
	bearer  string
	l       *logger.Logger
}

func (c *client) httpRequest(url string) (*http.Request, error) {
	url = fmt.Sprintf("https://%s/%s", c.apiHost, strings.TrimPrefix(url, "/"))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", c.bearer)

	return req, nil
}

func (c *client) getURL(url string) ([]byte, error) {
	req, err := c.httpRequest(url)
	if err != nil {
		return nil, err
	}

	c.l.Infof("kubernetes.client: getting URL: %s", req.URL.String())
	resp, err := c.httpC.Do(req)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(resp.Body)
}

func newClient(l *logger.Logger) (*client, error) {
	// This is a copy of the http.DefaultTransport
	// TODO(manugarg): Replace it by http.DefaultTransport.Clone() once we move to
	// Go 1.13.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	certs, err := ioutil.ReadFile(LocalCACert)
	if err != nil {
		return nil, fmt.Errorf("error while reading local ca.crt file: %v", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(certs)

	transport.TLSClientConfig = &tls.Config{
		RootCAs: caCertPool,
	}

	token, err := ioutil.ReadFile(LocalTokenFile)
	if err != nil {
		return nil, fmt.Errorf("error while reading local token file: %v", err)
	}

	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, fmt.Errorf("not running in cluster: KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT environment variables not set")
	}

	return &client{
		apiHost: net.JoinHostPort(host, port),
		httpC:   &http.Client{Transport: transport},
		bearer:  "Bearer " + string(token),
		l:       l,
	}, nil
}
