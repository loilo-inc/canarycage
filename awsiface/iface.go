package awsiface

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	alb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type (
	EcsClient interface {
		CreateService(ctx context.Context, params *ecs.CreateServiceInput, optFns ...func(*ecs.Options)) (*ecs.CreateServiceOutput, error)
		UpdateService(ctx context.Context, params *ecs.UpdateServiceInput, optFns ...func(*ecs.Options)) (*ecs.UpdateServiceOutput, error)
		DeleteService(ctx context.Context, params *ecs.DeleteServiceInput, optFns ...func(*ecs.Options)) (*ecs.DeleteServiceOutput, error)
		StartTask(ctx context.Context, params *ecs.StartTaskInput, optFns ...func(*ecs.Options)) (*ecs.StartTaskOutput, error)
		RegisterTaskDefinition(ctx context.Context, params *ecs.RegisterTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.RegisterTaskDefinitionOutput, error)
		DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
		DescribeTasks(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
		DescribeContainerInstances(ctx context.Context, params *ecs.DescribeContainerInstancesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeContainerInstancesOutput, error)
		ListTasks(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
		RunTask(ctx context.Context, params *ecs.RunTaskInput, optFns ...func(*ecs.Options)) (*ecs.RunTaskOutput, error)
		StopTask(ctx context.Context, params *ecs.StopTaskInput, optFns ...func(*ecs.Options)) (*ecs.StopTaskOutput, error)
		ListAttributes(ctx context.Context, params *ecs.ListAttributesInput, optFns ...func(*ecs.Options)) (*ecs.ListAttributesOutput, error)
		PutAttributes(ctx context.Context, params *ecs.PutAttributesInput, optFns ...func(*ecs.Options)) (*ecs.PutAttributesOutput, error)
		DescribeTaskDefinition(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error)
	}
	AlbClient interface {
		DescribeTargetGroups(ctx context.Context, params *alb.DescribeTargetGroupsInput, optFns ...func(*alb.Options)) (*alb.DescribeTargetGroupsOutput, error)
		DescribeTargetHealth(ctx context.Context, params *alb.DescribeTargetHealthInput, optFns ...func(*alb.Options)) (*alb.DescribeTargetHealthOutput, error)
		DescribeTargetGroupAttributes(ctx context.Context, params *alb.DescribeTargetGroupAttributesInput, optFns ...func(*alb.Options)) (*alb.DescribeTargetGroupAttributesOutput, error)
		RegisterTargets(ctx context.Context, params *alb.RegisterTargetsInput, optFns ...func(*alb.Options)) (*alb.RegisterTargetsOutput, error)
		DeregisterTargets(ctx context.Context, params *alb.DeregisterTargetsInput, optFns ...func(*alb.Options)) (*alb.DeregisterTargetsOutput, error)
	}
	Ec2Client interface {
		DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
		DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	}
)

var _ EcsClient = (*ecs.Client)(nil)
var _ AlbClient = (*alb.Client)(nil)
var _ Ec2Client = (*ec2.Client)(nil)
