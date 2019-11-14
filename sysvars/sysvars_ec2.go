// Copyright 2019 Didi Kohen
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
	"strings"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

func getEC2Meta(sysVars map[string]string) error {
	ec2Session, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("Could not create session %v", err)
	}
	ec2met := ec2metadata.New(ec2Session)
	// Doing the availability check in module since we need a session
	if ec2met.Available() == false {
		sysVars["EC2_METADATA_Availabile"] = "false"
		return nil
	}

	id, err := ec2met.GetInstanceIdentityDocument()
	if err != nil {
		sysVars["EC2_METADATA_Availabile"] = "false"
		return fmt.Errorf("Could not get instance document %v", err)
	}

	sysVars["EC2_METADATA_Availabile"] = "true"
	sysVars["EC2_DevpayProductCodes"] = strings.Join(id.DevpayProductCodes, ",")
	sysVars["EC2_MarketplaceProductCodes"] = strings.Join(id.MarketplaceProductCodes, ",")
	sysVars["EC2_AvailabilityZone"] = id.AvailabilityZone
	sysVars["EC2_PrivateIP"] = id.PrivateIP
	sysVars["EC2_Version"] = id.Version
	sysVars["EC2_Region"] = id.Region
	sysVars["EC2_InstanceID"] = id.InstanceID
	sysVars["EC2_BillingProducts"] = strings.Join(id.BillingProducts, ",")
	sysVars["EC2_InstanceType"] = id.InstanceType
	sysVars["EC2_AccountID"] = id.AccountID
	sysVars["EC2_PendingTime"] = id.PendingTime.String()
	sysVars["EC2_ImageID"] = id.ImageID
	sysVars["EC2_KernelID"] = id.KernelID
	sysVars["EC2_RamdiskID"] = id.RamdiskID
	sysVars["EC2_Architecture"] = id.Architecture
	return nil
}
