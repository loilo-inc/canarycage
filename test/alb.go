package test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

type AlbServer struct {
	*commons
}

type TargetGroup struct {
	Targets map[string]elbv2types.TargetDescription
}

func (ctx *AlbServer) DescribeTargetGroups(_ context.Context, input *elbv2.DescribeTargetGroupsInput, _ ...func(options *elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
	return &elbv2.DescribeTargetGroupsOutput{
		TargetGroups: []elbv2types.TargetGroup{
			{
				TargetGroupName:            aws.String("tgname"),
				TargetGroupArn:             aws.String(input.TargetGroupArns[0]),
				HealthyThresholdCount:      aws.Int32(1),
				HealthCheckIntervalSeconds: aws.Int32(0),
				LoadBalancerArns:           []string{"arn://hoge/app/aa/bb"},
			},
		},
	}, nil
}
func (ctx *AlbServer) DescribeTargetGroupAttributes(_ context.Context, input *elbv2.DescribeTargetGroupAttributesInput, _ ...func(options *elbv2.Options)) (*elbv2.DescribeTargetGroupAttributesOutput, error) {
	return &elbv2.DescribeTargetGroupAttributesOutput{
		Attributes: []elbv2types.TargetGroupAttribute{
			{
				Key:   aws.String("deregistration_delay.timeout_seconds"),
				Value: aws.String("0"),
			},
		},
	}, nil
}

func (ctx *AlbServer) DescribeTargetHealth(_ context.Context, input *elbv2.DescribeTargetHealthInput, _ ...func(options *elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
	var ret []elbv2types.TargetHealthDescription
	tg, ok := ctx.TargetGroups[*input.TargetGroupArn]
	if !ok {
		return nil, fmt.Errorf("target group not found")
	}
	for _, t := range input.Targets {
		_, ok := tg.Targets[*t.Id]
		var health elbv2types.TargetHealth
		if !ok {
			health = elbv2types.TargetHealth{
				State: elbv2types.TargetHealthStateEnumUnused,
			}
		} else {
			health = elbv2types.TargetHealth{
				State: elbv2types.TargetHealthStateEnumHealthy,
			}
		}
		ret = append(ret, elbv2types.TargetHealthDescription{
			Target:       &t,
			TargetHealth: &health,
		})
	}
	return &elbv2.DescribeTargetHealthOutput{
		TargetHealthDescriptions: ret,
	}, nil
}

func (ctx *AlbServer) RegisterTargets(_ context.Context, input *elbv2.RegisterTargetsInput, _ ...func(options *elbv2.Options)) (*elbv2.RegisterTargetsOutput, error) {
	tg, ok := ctx.TargetGroups[*input.TargetGroupArn]
	if !ok {
		tg = &TargetGroup{Targets: make(map[string]elbv2types.TargetDescription)}
		ctx.TargetGroups[*input.TargetGroupArn] = tg
	}
	for _, t := range input.Targets {
		tg.Targets[*t.Id] = t
	}
	return &elbv2.RegisterTargetsOutput{}, nil
}

func (ctx *AlbServer) DeregisterTargets(_ context.Context, input *elbv2.DeregisterTargetsInput, _ ...func(options *elbv2.Options)) (*elbv2.DeregisterTargetsOutput, error) {
	tg, ok := ctx.TargetGroups[*input.TargetGroupArn]
	if !ok {
		return nil, fmt.Errorf("target group not found")
	}
	for _, t := range input.Targets {
		delete(tg.Targets, *t.Id)
	}
	return &elbv2.DeregisterTargetsOutput{}, nil
}

func (ctx *AlbServer) DescribeInstances(_ context.Context, input *ec2.DescribeInstancesInput, _ ...func(options *ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
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
