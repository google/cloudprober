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
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	tlsconfigpb "github.com/google/cloudprober/common/tlsconfig/proto"
	cpb "github.com/google/cloudprober/rds/kubernetes/proto"
)

var testCACert = `
-----BEGIN CERTIFICATE-----
MIICVjCCAb+gAwIBAgIUY0TRq/rPKnOpZ+Bbv9hBMlKgiP0wDQYJKoZIhvcNAQEL
BQAwPTELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMSEwHwYDVQQKDBhJbnRlcm5l
dCBXaWRnaXRzIFB0eSBMdGQwHhcNMTkxMjE4MDA1MDMxWhcNMjAwMTE3MDA1MDMx
WjA9MQswCQYDVQQGEwJVUzELMAkGA1UECAwCQ0ExITAfBgNVBAoMGEludGVybmV0
IFdpZGdpdHMgUHR5IEx0ZDCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEA1z5z
InMY7e5AOgYnq9ACqwtIYRZZwEbBSy57Pe9kftEpUy5wfdN5YtBEiW+juy6CZLns
WKQ3sZ/gnbYvdRfkHnbTU6DZdt741H/YXMnbkT28In1NJYvz/FiJWiphUxbXNEmi
laHAbX+zkOnL81LX7NAArKlk0biK8iglW80oeRMCAwEAAaNTMFEwHQYDVR0OBBYE
FDHgRfu3kXFxPj0f4XYsxxukRYeKMB8GA1UdIwQYMBaAFDHgRfu3kXFxPj0f4XYs
xxukRYeKMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADgYEAI4xvCBkE
NbrbrZ4rPllcmYxgBrTvOUbaG8Lmhx1/qoOyx2LCca0pMQGB8cyMtT5/D7IZUlHk
5IG8ts/LpaDbRhTP4MQUCoN/FgAJ4e5Y33VWJocucgEFyv4aKl0Xgg+hO4ejpaxr
JeDPlMt+8OTQNVC63+SPzOvlUsOLFX74WMo=
-----END CERTIFICATE-----
`

func testFileWithContent(t *testing.T, content string) string {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Error creating temporary file for testing: %v", err)
		return ""
	}

	if _, err := f.Write([]byte(content)); err != nil {
		os.Remove(f.Name()) // clean up
		t.Fatalf("Error writing %s to temporary file: %s", content, f.Name())
		return ""
	}

	return f.Name()
}

func TestNewClientInCluster(t *testing.T) {
	cacrtF := testFileWithContent(t, testCACert)
	defer os.Remove(cacrtF) // clean up

	oldLocalCACert := LocalCACert
	LocalCACert = cacrtF
	defer func() { LocalCACert = oldLocalCACert }()

	testToken := "test-token"
	tokenF := testFileWithContent(t, testToken)
	defer os.Remove(tokenF) // clean up

	oldLocalTokenFile := LocalTokenFile
	LocalTokenFile = tokenF
	defer func() { LocalTokenFile = oldLocalTokenFile }()

	os.Setenv("KUBERNETES_SERVICE_HOST", "test-api-host")
	os.Setenv("KUBERNETES_SERVICE_PORT", "4123")

	tc, err := newClient(&cpb.ProviderConfig{}, nil)
	if err != nil {
		t.Fatalf("error while creating new client: %v", err)
	}

	expectedAPIHost := "test-api-host:4123"
	if tc.apiHost != "test-api-host:4123" {
		t.Errorf("client.apiHost: got=%s, exepcted=%s", tc.apiHost, expectedAPIHost)
	}

	expectedToken := "Bearer " + testToken
	if tc.bearer != expectedToken {
		t.Errorf("client.bearer: got=%s, exepcted=%s", tc.bearer, expectedToken)
	}

	if tc.httpC == nil || tc.httpC.Transport.(*http.Transport).TLSClientConfig == nil {
		t.Errorf("Client's HTTP client or TLS config are unexpectedly nil.")
	}
}

func TestNewClientWithTLS(t *testing.T) {
	cacrtF := testFileWithContent(t, testCACert)
	defer os.Remove(cacrtF) // clean up

	oldLocalCACert := LocalCACert
	LocalCACert = cacrtF
	defer func() { LocalCACert = oldLocalCACert }()

	testAPIServerAddr := "test-api-server-addr"
	tc, err := newClient(&cpb.ProviderConfig{
		ApiServerAddress: &testAPIServerAddr,
		TlsConfig: &tlsconfigpb.TLSConfig{
			CaCertFile: proto.String(cacrtF),
		},
	}, nil)
	if err != nil {
		t.Fatalf("error while creating new client: %v", err)
	}

	if tc.apiHost != testAPIServerAddr {
		t.Errorf("client.apiHost: got=%s, exepcted=%s", tc.apiHost, testAPIServerAddr)
	}

	if tc.bearer != "" {
		t.Errorf("client.bearer: got=%s, exepcted=''", tc.bearer)
	}

	if tc.httpC == nil || tc.httpC.Transport.(*http.Transport).TLSClientConfig == nil {
		t.Errorf("Client's HTTP client or TLS config are unexpectedly nil.")
	}
}
