// Copyright 2017 Google Inc.
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

// Package rtcservice provides utility functions for the cloudprober project
// used when dealing with the Runtime Configurator API.
// https://cloud.google.com/deployment-manager/runtime-configurator/reference/rest/
// This API essentially provides a small key/val store for GCE
// projects. Cloudprober uses this for things like maintaining a lameduck list,
// listing probing hosts in the projects, and in general any shared state that
// cloudprober instances might require.
//
// Because RTC requires a GCE project, this package provides all functionality
// as methods on the Config interface. This allows the behavior to be more
// easily mocked for testing.
//
// rtcservice.Config is meant to represent a configuration or resource in the
// RTC sense (see https://cloud.google.com/deployment-manager/runtime-configurator/).
// If one needs to interact with multiple configurations, they will need multiple
// instances of rtc.Config.
package rtcservice

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	runtimeconfig "google.golang.org/api/runtimeconfig/v1beta1"
)

// The Config interface provides communication to the Runtime Configurator API.
// It represents a single runtime configuration, allowing one to read and write
// variables to the configuration.
type Config interface {
	// GetProject returns the project name for the configuration
	GetProject() string
	// Write adds or changes a key value pair to a configuration.
	Write(key string, val []byte) error
	// Delete removes a variable from a configuration.
	Delete(key string) error
	// Val returns the Value stored by a variable.
	Val(v *runtimeconfig.Variable) ([]byte, error)
	// List lists all variables in a configuration.
	List() ([]*runtimeconfig.Variable, error)
	// FilterList will list all variables in a configuration, filtering variable
	// names by the filter string. This mirrors the behavior found in
	// https://cloud.google.com/deployment-manager/runtime-configurator/reference/rest/v1beta1/projects.configs.variables/list
	FilterList(filter string) ([]*runtimeconfig.Variable, error)
}

// Full-featured implementation of Rtc interface. Only accessible through the
// New function.
type impl struct {
	svc  *runtimeconfig.ProjectsConfigsVariablesService
	proj string
	cfg  string
}

// New provides an interface to the RTC API for a given project. The string proj
// represents the project name (such as "google.com:bbmc-test"), and the string
// "cfg" will represent the name of the RTC resource.
//
// In order to provide fail-fast sanitation, New will check that the provided
// project string and cfg string are reachable. If not, an error will be returned.
// Note that this means New cannot be used to establish a new RTC configuration ---
// the configuration must already exist.
//
// New also takes an OAuth2.0 enabled *http.Client for API access. If a nil
// *http.Client is provided, a new http.Client is created using
// google.DefaultClient, which uses default credentials.
func New(proj string, cfg string, c *http.Client) (Config, error) {
	svc, err := getService(c)
	if err != nil {
		return nil, err
	}
	// TODO: Consider checking for the configuration errors before returning.
	return &impl{svc, proj, cfg}, nil
}

// This helper function is used to actually connect to an RTC client.
func getService(c *http.Client) (*runtimeconfig.ProjectsConfigsVariablesService, error) {
	if c == nil {
		var err error
		c, err = google.DefaultClient(context.TODO(), runtimeconfig.CloudruntimeconfigScope)
		if err != nil {
			return nil, err
		}
	}
	rtcService, err := runtimeconfig.New(c)
	if err != nil {
		return nil, err
	}
	return runtimeconfig.NewProjectsConfigsVariablesService(rtcService), nil
}

// GetProject will return the alphanumeric project ID for the configuration
func (s *impl) GetProject() string {
	return s.proj
}

// Write will add or change key/val pair in the configuration for s's project
// using the RTC API. An empty key will return an error, however empty vals are
// perfectly fine, and will return a variable with an empty-string value.
func (s *impl) Write(key string, val []byte) error {
	path := "projects/" + s.proj + "/configs/" + s.cfg
	encoded := base64.StdEncoding.EncodeToString(val)
	v := runtimeconfig.Variable{
		Name:  path + "/variables/" + key,
		Value: encoded,
		// ForceSendFields will force value to be sent, even if empty.
		ForceSendFields: []string{"value"},
	}

	// Create. If error, check error type, and maybe update.
	_, err := s.svc.Create(path, &v).Do()
	if err != nil {
		// If error is 'ALREADY_EXISTS', then we call update.
		apierr, ok := err.(*googleapi.Error)
		if ok && apierr.Message == "Requested entity already exists" {
			path += "/variables/" + key
			v.Name = ""
			_, err = s.svc.Update(path, &v).Do()
			if err != nil {
				return fmt.Errorf("rtc.Write %#v : unable to update variable : %v", v, err)
			}
		} else {
			return fmt.Errorf("rtc.Write %#v : unable to create variable : %v", v, err)
		}
	}
	return err
}

// Delete removes a key/val pair from the configuration in s's project using
// the RTC API. An empty key will return an error.
func (s *impl) Delete(key string) error {
	path := "projects/" + s.proj + "/configs/" + s.cfg + "/variables/" + key
	_, err := s.svc.Delete(path).Do()
	return err
}

// Val attempts to decode the value for a given Variable.
func (s *impl) Val(v *runtimeconfig.Variable) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(v.Value)
	if err != nil {
		return nil, fmt.Errorf("rtc.Val %#v : unable to decode value : %v", v, err)
	}
	return decoded, nil
}

// List provides a slice of all variables in the configuration for s's
// project using the RTC API.
func (s *impl) List() ([]*runtimeconfig.Variable, error) {
	path := "projects/" + s.proj + "/configs/" + s.cfg
	resp, err := s.svc.List(path).ReturnValues(true).Do()
	if err != nil {
		return nil, err
	}
	return resp.Variables, nil
}

// FilterList provides a slice of all variables in the configuration for s's
// project using the RTC API, filtered by the given filter string. More about this
// behavior can be found in the RTC documentation.
// https://cloud.google.com/deployment-manager/runtime-configurator/reference/rest/v1beta1/projects.configs.variables/list
func (s *impl) FilterList(filter string) ([]*runtimeconfig.Variable, error) {
	path := "projects/" + s.proj + "/configs/" + s.cfg
	resp, err := s.svc.List(path).Filter(filter).ReturnValues(true).Do()
	if err != nil {
		return nil, err
	}
	return resp.Variables, nil
}
