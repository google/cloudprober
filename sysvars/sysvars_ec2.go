// Copyright 2019-2020 The Cloudprober Authors.
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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/google/cloudprober/logger"
)

var ec2Vars = func(sysVars map[string]string, l *logger.Logger) (bool, error) {
	s, err := session.NewSession(&aws.Config{
		MaxRetries: aws.Int(0),
	})
	if err != nil {
		// We ignore session errors. It's not clear what can cause them.
		l.Warningf("sysvars_ec2: could not create AWS session: %v", err)
		return false, nil
	}

	md := ec2metadata.New(s)
	// Doing the availability check in module since we need a session
	if md.Available() == false {
		return false, nil
	}

	id, err := md.GetInstanceIdentityDocument()
	if err != nil {
		sysVars["EC2_METADATA_Available"] = "false"
		return true, fmt.Errorf("sysvars_ec2: could not get instance identity document %v", err)
	}

	sysVars["EC2_METADATA_Available"] = "true"
	sysVars["EC2_AvailabilityZone"] = id.AvailabilityZone
	sysVars["EC2_PrivateIP"] = id.PrivateIP
	sysVars["EC2_Region"] = id.Region
	sysVars["EC2_InstanceID"] = id.InstanceID
	sysVars["EC2_InstanceType"] = id.InstanceType
	sysVars["EC2_ImageID"] = id.ImageID
	sysVars["EC2_KernelID"] = id.KernelID
	sysVars["EC2_RamdiskID"] = id.RamdiskID
	sysVars["EC2_Architecture"] = id.Architecture
	return true, nil
}
