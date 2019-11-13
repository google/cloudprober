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
		fmt.Printf("Could not create session %v", err)
		return err
	}
	ec2met := ec2metadata.New(ec2Session)
	// Doing the availability check in module since we need a session
	if ec2met.Available() == false {
		sysVars["EC2_METADATA_Availabile"] = "false"
		return nil
	}

	id, err := ec2met.GetInstanceIdentityDocument()
	if err != nil {
		fmt.Printf("Could not get instance document %v", err)
		return err
	}

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
