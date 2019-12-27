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

package oauth

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	configpb "github.com/google/cloudprober/common/oauth/proto"
	"google3/base/go/log"
)

func createTempFile(t *testing.T, b []byte) string {
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
		return ""
	}

	defer tmpfile.Close()
	if _, err := tmpfile.Write(b); err != nil {
		log.Fatal(err)
	}

	return tmpfile.Name()
}

var testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIBPAIBAAJBAN6ErRPkzBWt+R+kMtbbAgmFal+ZbVoWSJDzsLFwdzfy0QaI6Svq
g4zpfn/H7lXi5MPJ+OWWhFy2DRD0L01PF8kCAwEAAQJBAM59MF+Vog08NEI4jTT0
Zx+OvveX2PIQW6anfQAr7XXsEo910bPjb9YdfFaHyQCS8aIYeQ7vXD8tV6Vlu93B
LkECIQD77dd8JWEZp2ZCt0SpN6mPcNOvVoXhvdKp9SiqMorn0wIhAOIdK5hbx/d+
rextXrNWAeT2PrWxYN7FX1neuAzebrxzAiEA9b9vuQlZa8XwqdnOX2cNvv+nbt1u
4eLiMaoVDdkZyMMCIQDRXslbTsEevsI1RiCGVoFyjUEL5K8aGBBumvg5kk1fWQIg
X5mM0KsPPfa9wLSGj6CPt2c3skhQu/k2FjMASmaQbZw=
-----END RSA PRIVATE KEY-----`

func testJSONKey() string {
	keyTmpl := `{
  "type": "service_account",
  "project_id": "cloud-nat-prober",
  "private_key_id": "testprivateid",
  "private_key": "%s",
  "client_email": "test-consumer@test-project.iam.gserviceaccount.com",
  "client_id": "testclientid",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test-consumer%%40test-project.iam.gserviceaccount.com"
}
`
	return fmt.Sprintf(keyTmpl, strings.Replace(testPrivateKey, "\n", "\\n", -1))
}

func TestGoogleCredentials(t *testing.T) {
	jsonF := createTempFile(t, []byte(testJSONKey()))

	googleC := &configpb.GoogleCredentials{
		JsonFile: proto.String(jsonF),
	}

	c := &configpb.Config{
		Type: &configpb.Config_GoogleCredentials{
			GoogleCredentials: googleC,
		},
	}

	_, err := TokenSourceFromConfig(c, nil)
	if err != nil {
		t.Errorf("Config: %v, Unexpected error: %v", c, err)
	}

	// Set audience, it should fail as jwt_as_access_token is not set.
	googleC.Audience = proto.String("test-audience")

	_, err = TokenSourceFromConfig(c, nil)
	if err == nil {
		t.Errorf("Config: %v, Expected error, but got none.", c)
	}

	// Set jwt_as_access_token, no errors now.
	googleC.JwtAsAccessToken = proto.Bool(true)

	_, err = TokenSourceFromConfig(c, nil)
	if err != nil {
		t.Errorf("Config: %v, Unexpected error: %v", c, err)
	}
}
