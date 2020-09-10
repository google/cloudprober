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

// Package tlsconfig implements utilities to parse TLSConfig.
package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/google/cloudprober/common/file"
	configpb "github.com/google/cloudprober/common/tlsconfig/proto"
)

// UpdateTLSConfig parses the provided protobuf and updates the tls.Config object.
func UpdateTLSConfig(tlsConfig *tls.Config, c *configpb.TLSConfig, addClientCACerts bool) error {
	if c.GetDisableCertValidation() {
		tlsConfig.InsecureSkipVerify = true
	}

	if c.GetCaCertFile() != "" {
		caCert, err := file.ReadFile(c.GetCaCertFile())
		if err != nil {
			return fmt.Errorf("common/tlsconfig: error reading CA cert file (%s): %v", c.GetCaCertFile(), err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("error while adding CA certs from: %s", c.GetCaCertFile())
		}

		tlsConfig.RootCAs = caCertPool
		// Client CA certs are used by servers to authenticate clients.
		if addClientCACerts {
			tlsConfig.ClientCAs = caCertPool
		}
	}

	if c.GetTlsCertFile() != "" {
		certPEMBlock, err := file.ReadFile(c.GetTlsCertFile())
		if err != nil {
			return fmt.Errorf("common/tlsconfig: error reading TLS cert file (%s): %v", c.GetTlsCertFile(), err)
		}
		keyPEMBlock, err := file.ReadFile(c.GetTlsKeyFile())
		if err != nil {
			return fmt.Errorf("common/tlsconfig: error reading TLS key file (%s): %v", c.GetTlsKeyFile(), err)
		}

		cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			return fmt.Errorf("common/tlsconfig: error initializing cert from cert key pair: %v", err)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}

	if c.GetServerName() != "" {
		tlsConfig.ServerName = c.GetServerName()
	}

	return nil
}
