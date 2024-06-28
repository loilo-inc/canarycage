package test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Ec2Server struct {
	*commons
}

func (c *Ec2Server) DescribeSubnets(_ context.Context, input *ec2.DescribeSubnetsInput, _ ...func(options *ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return &ec2.DescribeSubnetsOutput{
		Subnets: []ec2types.Subnet{{
			AvailabilityZone:              aws.String("us-west-2"),
			AvailabilityZoneId:            nil,
			AvailableIpAddressCount:       nil,
			CidrBlock:                     nil,
			CustomerOwnedIpv4Pool:         nil,
			DefaultForAz:                  nil,
			EnableDns64:                   nil,
			EnableLniAtDeviceIndex:        nil,
			Ipv6CidrBlockAssociationSet:   nil,
			Ipv6Native:                    nil,
			MapCustomerOwnedIpOnLaunch:    nil,
			MapPublicIpOnLaunch:           nil,
			OutpostArn:                    nil,
			OwnerId:                       nil,
			PrivateDnsNameOptionsOnLaunch: nil,
			State:                         ec2types.SubnetStateAvailable,
			SubnetArn:                     nil,
			SubnetId:                      aws.String("subnet-1234567890abcdefg"),
			Tags:                          nil,
			VpcId:                         nil,
		}},
	}, nil
}

func (c *Ec2Server) DescribeInstances(_ context.Context, input *ec2.DescribeInstancesInput, _ ...func(options *ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{
		Reservations: []ec2types.Reservation{{
			Instances: []ec2types.Instance{{
				InstanceId:       aws.String("i-123456"),
				PrivateIpAddress: aws.String("127.0.1.0"),
				SubnetId:         aws.String("us-west-2a"),
			}},
		}},
	}, nil
}
