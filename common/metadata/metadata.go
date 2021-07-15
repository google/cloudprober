// Copyright 2021 The Cloudprober Authors.
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
Package metadata implements metadata related utilities.
*/
package metadata

import (
	"io/ioutil"
	"os"
)

// IsKubernetes return true if running on Kubernetes.
// It uses the environment variable KUBERNETES_SERVICE_HOST to decide if we
// we are running Kubernetes.
func IsKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// KubernetesNamespace returns the Kubernetes namespace. It returns an empty
// string if there is an error in retrieving the namespace.
func KubernetesNamespace() string {
	namespaceBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		return string(namespaceBytes)
	}
	return ""
}
