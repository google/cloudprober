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
	"os"

	"cloud.google.com/go/compute/metadata"
	md "github.com/google/cloudprober/common/metadata"
	monitoring "google.golang.org/api/monitoring/v3"
)

func kubernetesResource(projectID string) (*monitoring.MonitoredResource, error) {
	namespace := md.KubernetesNamespace()

	// We ignore error in getting cluster name and location. These attributes
	// should be set while running on GKE.
	clusterName, _ := metadata.InstanceAttributeValue("cluster-name")
	location, _ := metadata.InstanceAttributeValue("cluster-location")

	// We can likely use cluster-location instance attribute for location. Using
	// zone provides more granular scope though.
	return &monitoring.MonitoredResource{
		Type: "k8s_container",
		Labels: map[string]string{
			"cluster_name":   clusterName,
			"location":       location,
			"project_id":     projectID,
			"pod_name":       os.Getenv("HOSTNAME"),
			"namespace_name": namespace,
			// To get the `container_name` label, users need to explicitly provide it.
			"container_name": os.Getenv("CONTAINER_NAME"),
		},
	}, nil
}

func gceResource(projectID string) (*monitoring.MonitoredResource, error) {
	name, err := metadata.InstanceName()
	if err != nil {
		return nil, err
	}

	zone, err := metadata.Zone()
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
	if md.IsKubernetes() {
		return kubernetesResource(projectID)
	}
	return gceResource(projectID)
}
