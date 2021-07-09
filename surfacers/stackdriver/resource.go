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

package stackdriver

import (
	"io/ioutil"
	"os"

	"cloud.google.com/go/compute/metadata"
	monitoring "google.golang.org/api/monitoring/v3"
)

func isKubernetesEngine() (bool, string) {
	clusterName, err := metadata.InstanceAttributeValue("cluster-name")
	// Note: InstanceAttributeValue can return "", nil
	if err != nil || clusterName == "" {
		return false, ""
	}
	return true, clusterName
}

func kubernetesResource(clusterName, projectID, zone string) (*monitoring.MonitoredResource, error) {
	namespaceBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	namespaceName := ""
	if err == nil {
		namespaceName = string(namespaceBytes)
	}

	// We can likely use cluster-location instance attribute for location. Using
	// zone provides more granular scope though.
	return &monitoring.MonitoredResource{
		Type: "k8s_container",
		Labels: map[string]string{
			"cluster_name":   clusterName,
			"location":       zone,
			"project_id":     projectID,
			"pod_name":       os.Getenv("HOSTNAME"),
			"namespace_name": namespaceName,
			// To get the `container_name` label, users need to explicitly provide it.
			"container_name": os.Getenv("CONTAINER_NAME"),
		},
	}, nil
}

func gceResource(projectID, zone string) (*monitoring.MonitoredResource, error) {
	name, err := metadata.InstanceName()
	if err != nil {
		return nil, err
	}

	return &monitoring.MonitoredResource{
		Type: "gce_instance",
		Labels: map[string]string{
			"project_id": projectID,
			// Note that following is not correct, as we are adding instance name as
			// instance id, but this is what we have been doing all along, and for
			// monitoring instance name may be more useful, at least as of now there
			// is no automatic instance to custom-monitoring mapping.
			"instance_id": name,
			"zone":        zone,
		},
	}, nil
}

func monitoredResourceOnGCE(projectID string) (*monitoring.MonitoredResource, error) {
	zone, err := metadata.Zone()
	if err != nil {
		return nil, err
	}
	if ok, clusterName := isKubernetesEngine(); ok {
		return kubernetesResource(clusterName, projectID, zone)
	}
	return gceResource(projectID, zone)
}
