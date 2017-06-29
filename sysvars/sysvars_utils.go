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

package sysvars

import (
	"fmt"
	"net"
	"os"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
)

// maxNICs is the number of NICs allowed on a VM. Used by addGceNicInfo.
var maxNICs = 8

func gceVars(vars map[string]string) error {
	for _, k := range []string{
		"zone",
		"project",
		"project_id",
		"instance",
		"instance_id",
		"internal_ip",
		"external_ip",
		"instance_template",
	} {
		var v string
		var err error
		switch k {
		case "zone":
			v, err = metadata.Zone()
		case "project":
			v, err = metadata.ProjectID()
		case "project_id":
			v, err = metadata.NumericProjectID()
		case "instance":
			v, err = metadata.InstanceName()
		case "instance_id":
			v, err = metadata.InstanceID()
		case "internal_ip":
			v, err = metadata.InternalIP()
		case "external_ip":
			v, err = metadata.ExternalIP()
		case "instance_template":
			// `instance_template` may not be defined, depending on how cloudprober
			// was deployed. If an error is returned when fetching the metadata,
			// just fall back "undefined".
			v, err = metadata.InstanceAttributeValue("instance-template")
			if err != nil {
				glog.Infof("No instance_template found. Defaulting to undefined.")
				v = "undefined"
				err = nil
			} else {
				tokens := strings.Split(v, "/")
				v = tokens[len(tokens)-1]
			}
		default:
			return fmt.Errorf("utils.GCEVars: unknown variable key %q", k)
		}
		if err != nil {
			return fmt.Errorf("utils.GCEVars: error while getting %s from metadata: %v", k, err)
		}
		vars[k] = v
	}
	zoneParts := strings.Split(vars["zone"], "-")
	vars["region"] = strings.Join(zoneParts[0:len(zoneParts)-1], "-")
	addGceNicInfo(vars)
	return nil
}

// addGceNicInfo adds nic information to vars.
// The following information is added for each nic.
// - Primary IP, if one is assigned to nic.
// - External IP, if one is assigned to nic.
// - An IP alias, if any IP alias ranges are assigned to nic.
//
// See the following document for more information on metadata.
// https://cloud.google.com/compute/docs/storing-retrieving-metadata
func addGceNicInfo(vars map[string]string) {
	for i := 0; i < maxNICs; i++ {
		k := fmt.Sprintf("instance/network-interfaces/%v/ip", i)
		v, err := metadata.Get(k)
		// If there is no private IP for NIC, NIC doesn't exist.
		if err != nil {
			continue
		}
		vars[k] = v

		k = fmt.Sprintf("instance/network-interfaces/%v/access-configs/%v/external-ip", i, i)
		v, err = metadata.Get(k)
		// NIC may exist but not have external IP.
		if err != nil {
			continue
		}
		vars[k] = v

		k = fmt.Sprintf("instance/network-interfaces/%v/ip-aliases/0", i)
		v, err = metadata.Get(k)
		// NIC may not have any IP alias ranges.
		if err != nil {
			continue
		}
		// Extract a sample IP address from the IP range returned via above metadata query.
		ip, _, _ := net.ParseCIDR(v)
		vars[k+"-sample"] = ip.String()
	}
}

// SystemVars finds system level variables and adds them to the map passed in the arguments.
func SystemVars(vars map[string]string) error {
	if vars == nil {
		vars = make(map[string]string)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("utils.SystemVars: error getting local hostname: %v", err)
	}
	vars["hostname"] = hostname
	if metadata.OnGCE() {
		return gceVars(vars)
	}
	return nil
}
