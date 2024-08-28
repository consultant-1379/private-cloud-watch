// Copyright 2015 Apcera Inc. All rights reserved.

package aws

import (
	"fmt"
	"os"

	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/apcera/util/uuid"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/session"
	"github.com/apcera/libretto/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
)

// ValidCredentials sends a dummy request to AWS to check if credentials are
// valid. An error is returned if credentials are missing or region is missing.
func ValidCredentials(region string) error {
	_, err := getService(region).DescribeInstances(nil)
	awsErr, isAWS := err.(awserr.Error)
	if !isAWS {
		return err
	}

	switch awsErr.Code() {
	case noCredsCode:
		return ErrNoCreds
	case noRegionCode:
		return ErrNoRegion
	}

	return nil
}

func getInstanceVolumeIDs(svc *ec2.EC2, instID string) ([]string, error) {
	resp, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("attachment.instance-id"),
				Values: []*string{aws.String(instID)}},
		},
	})
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		if v == nil || v.VolumeId == nil {
			continue
		}

		ids = append(ids, *v.VolumeId)
	}

	return ids, nil
}

func getNonRootDeviceNames(svc *ec2.EC2, instID string) ([]string, error) {
	resp, err := svc.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
		Attribute:  aws.String("blockDeviceMapping"),
		InstanceId: aws.String(instID),
	})
	if err != nil {
		return nil, err
	}

	var rootDevice string
	if resp.RootDeviceName != nil && resp.RootDeviceName.Value != nil {
		rootDevice = *resp.RootDeviceName.Value
	}

	names := make([]string, 0, len(resp.BlockDeviceMappings))
	for _, m := range resp.BlockDeviceMappings {
		if m == nil || m.DeviceName == nil {
			continue
		}

		if *m.DeviceName == rootDevice {
			continue
		}

		names = append(names, *m.DeviceName)
	}

	return names, nil
}

func setNonRootDeleteOnDestroy(svc *ec2.EC2, instID string, delOnTerm bool) error {
	devNames, err := getNonRootDeviceNames(svc, instID)
	if err != nil {
		return fmt.Errorf("DescribeInstanceAttribute: %s", err)
	}

	devices := make([]*ec2.InstanceBlockDeviceMappingSpecification, 0, len(devNames))
	for _, name := range devNames {
		devices = append(devices, &ec2.InstanceBlockDeviceMappingSpecification{
			DeviceName: aws.String(name),
			Ebs: &ec2.EbsInstanceBlockDeviceSpecification{
				DeleteOnTermination: aws.Bool(delOnTerm),
			},
		})
	}

	_, err = svc.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		InstanceId:          aws.String(instID),
		BlockDeviceMappings: devices,
	})
	if err != nil {
		return fmt.Errorf("ModifyInstanceAttribute: %s", err)
	}

	return nil
}

func getService(region string) *ec2.EC2 {
	creds := credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.EnvProvider{},               // check environment
			&credentials.SharedCredentialsProvider{}, // check home dir
		},
	)

	if region == "" { // user didn't set region
		region = os.Getenv("AWS_DEFAULT_REGION") // aws cli checks this
		if region == "" {
			region = os.Getenv("AWS_REGION") // aws sdk checks this
		}
	}

	return ec2.New(session.New(&aws.Config{
		Credentials: creds,
		Region:      &region,
	}))
}

func instanceInfo(vm *VM) *ec2.RunInstancesInput {
	if vm.Name == "" {
		vm.Name = fmt.Sprintf("libretto-vm-%s", uuid.Variant4())
	}
	if vm.AMI == "" {
		vm.AMI = defaultAMI
	}
	if vm.InstanceType == "" {
		vm.InstanceType = defaultInstanceType
	}
	if vm.VolumeSize == 0 {
		vm.VolumeSize = defaultVolumeSize
	}
	if vm.VolumeType == "" {
		vm.VolumeType = defaultVolumeType
	}

	var sid *string
	if vm.Subnet != "" {
		sid = aws.String(vm.Subnet)
	}

	var sgid []*string
	if vm.SecurityGroup != "" {
		sgid = make([]*string, 1)
		sgid[0] = aws.String(vm.SecurityGroup)
	}

	return &ec2.RunInstancesInput{
		ImageId:      aws.String(vm.AMI),
		InstanceType: aws.String(vm.InstanceType),
		KeyName:      aws.String(vm.KeyPair),
		MaxCount:     aws.Int64(instanceCount),
		MinCount:     aws.Int64(instanceCount),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{DeviceName: aws.String(vm.DeviceName),
				Ebs: &ec2.EbsBlockDevice{
					VolumeSize:          aws.Int64(int64(vm.VolumeSize)),
					VolumeType:          aws.String(vm.VolumeType),
					DeleteOnTermination: aws.Bool(!vm.KeepRootVolumeOnDestroy),
				}},
		},
		Monitoring: &ec2.RunInstancesMonitoringEnabled{
			Enabled: aws.Bool(true),
		},
		SubnetId:         sid,
		SecurityGroupIds: sgid,
	}
}

func hasInstanceID(instance *ec2.Instance) bool {
	if instance == nil || instance.InstanceId == nil {
		return false
	}

	return true
}
