package test

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"go.uber.org/mock/gomock"
)

func Setup(ctrl *gomock.Controller, envars *env.Envars, currentTaskCount int, launchType ecstypes.LaunchType) (
	*MockContext,
	*mock_awsiface.MockEcsClient,
	*mock_awsiface.MockAlbClient,
	*mock_awsiface.MockEc2Client,
) {
	mocker := NewMockContext()

	ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
	ecsMock.EXPECT().CreateService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.CreateService).AnyTimes()
	ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.UpdateService).AnyTimes()
	ecsMock.EXPECT().DeleteService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.DeleteService).AnyTimes()
	ecsMock.EXPECT().StartTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.StartTask).AnyTimes()
	ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.RegisterTaskDefinition).AnyTimes()
	ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.DescribeServices).AnyTimes()
	ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.DescribeTasks).AnyTimes()
	ecsMock.EXPECT().ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.ListTasks).AnyTimes()
	ecsMock.EXPECT().DescribeContainerInstances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.DescribeContainerInstances).AnyTimes()
	ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.RunTask).AnyTimes()
	ecsMock.EXPECT().StopTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.StopTask).AnyTimes()

	albMock := mock_awsiface.NewMockAlbClient(ctrl)
	albMock.EXPECT().DescribeTargetGroups(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Alb.DescribeTargetGroups).AnyTimes()
	albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Alb.DescribeTargetHealth).AnyTimes()
	albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Alb.DescribeTargetGroupAttributes).AnyTimes()
	albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Alb.RegisterTargets).AnyTimes()
	albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Alb.DeregisterTargets).AnyTimes()

	ec2Mock := mock_awsiface.NewMockEc2Client(ctrl)
	ec2Mock.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ec2.DescribeSubnets).AnyTimes()
	ec2Mock.EXPECT().DescribeInstances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ec2.DescribeInstances).AnyTimes()
	td, _ := mocker.Ecs.RegisterTaskDefinition(context.Background(), envars.TaskDefinitionInput)
	if currentTaskCount >= 0 {
		input := *envars.ServiceDefinitionInput
		input.TaskDefinition = td.TaskDefinition.TaskDefinitionArn
		input.DesiredCount = aws.Int32(int32(currentTaskCount))
		input.LaunchType = launchType
		svc, _ := mocker.Ecs.CreateService(context.Background(), &input)
		if len(svc.Service.LoadBalancers) > 0 {
			_, _ = mocker.Alb.RegisterTargets(context.Background(), &elbv2.RegisterTargetsInput{
				TargetGroupArn: svc.Service.LoadBalancers[0].TargetGroupArn,
			})
		}
	}
	return mocker, ecsMock, albMock, ec2Mock
}

func DefaultEnvars() *env.Envars {
	service := &ecs.CreateServiceInput{
		Cluster:        aws.String("cluster"),
		ServiceName:    aws.String("service"),
		TaskDefinition: aws.String("-"),
		LoadBalancers: []ecstypes.LoadBalancer{
			{TargetGroupArn: aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/test/123456789012"),
				ContainerName:    aws.String("container"),
				ContainerPort:    aws.Int32(8000),
				LoadBalancerName: aws.String("lb"),
			},
		},
		DesiredCount:    aws.Int32(1),
		LaunchType:      ecstypes.LaunchTypeFargate,
		PlatformVersion: aws.String("1.4.0"),
		Role:            aws.String("arn:aws:iam::123456789012:role/ecsServiceRole"),
		DeploymentConfiguration: &ecstypes.DeploymentConfiguration{
			MaximumPercent:        aws.Int32(200),
			MinimumHealthyPercent: aws.Int32(100),
		},
		NetworkConfiguration: &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        []string{"subnet-12345678"},
				SecurityGroups: []string{"sg-12345678"},
				AssignPublicIp: ecstypes.AssignPublicIpDisabled,
			},
		},
		HealthCheckGracePeriodSeconds: aws.Int32(0),
		SchedulingStrategy:            ecstypes.SchedulingStrategyReplica,
	}
	taskDefinition := &ecs.RegisterTaskDefinitionInput{
		Family:           aws.String("test-task"),
		TaskRoleArn:      aws.String("arn:aws:iam::123456789012:role/ecsTaskExecutionRole"),
		ExecutionRoleArn: aws.String("arn:aws:iam::123456789012:role/ecsTaskExecutionRole"),
		NetworkMode:      ecstypes.NetworkModeAwsvpc,
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name: aws.String("container"),
				PortMappings: []ecstypes.PortMapping{
					{ContainerPort: aws.Int32(8000), HostPort: aws.Int32(80)},
				},
				Essential: aws.Bool(true),
				HealthCheck: &ecstypes.HealthCheck{
					Command: []string{"CMD-SHELL", "curl -f http://localhost:8000/ || exit 1"},
				},
			},
			{
				Name: aws.String("containerWithoutHealthCheck"),
				PortMappings: []ecstypes.PortMapping{
					{ContainerPort: aws.Int32(8000), HostPort: aws.Int32(81)},
				},
			},
		},
		RequiresCompatibilities: []ecstypes.Compatibility{ecstypes.CompatibilityFargate},
		Cpu:                     aws.String("256"),
		Memory:                  aws.String("512"),
	}
	return &env.Envars{
		Region:                 "us-west-2",
		Cluster:                "cage-test",
		Service:                "service",
		ServiceDefinitionInput: service,
		TaskDefinitionInput:    taskDefinition,
	}
}

func ReadServiceDefinition(path string) *ecs.CreateServiceInput {
	d, _ := os.ReadFile(path)
	var dest ecs.CreateServiceInput
	if err := json.Unmarshal(d, &dest); err != nil {
		log.Fatalf(err.Error())
	}
	return &dest
}
