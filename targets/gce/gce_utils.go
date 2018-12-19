package gce

import (
	"context"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// defaultComputeService returns the compute service to use for the GCE API
// calls.
func defaultComputeService(apiVersion string) (*compute.Service, error) {
	client, err := google.DefaultClient(context.TODO(), compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	cs, err := compute.New(client)
	if err != nil {
		return nil, err
	}
	cs.BasePath = "https://www.googleapis.com/compute/" + apiVersion + "/projects/"
	return cs, nil
}
