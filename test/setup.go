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
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
)

func Setup(ctrl *gomock.Controller, envars *env.Envars, currentTaskCount int, launchType ecstypes.LaunchType) (
	*MockContext,
	*mock_awsiface.MockEcsClient,
	*mock_awsiface.MockAlbClient,
	*mock_awsiface.MockEc2Client,
) {
	mocker := NewMockContext()

	ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
	ecsMock.EXPECT().CreateService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.CreateService).AnyTimes()
	ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.UpdateService).AnyTimes()
	ecsMock.EXPECT().DeleteService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeleteService).AnyTimes()
	ecsMock.EXPECT().StartTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.StartTask).AnyTimes()
	ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTaskDefinition).AnyTimes()
	ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeServices).AnyTimes()
	ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTasks).AnyTimes()
	ecsMock.EXPECT().ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.ListTasks).AnyTimes()
	ecsMock.EXPECT().DescribeContainerInstances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeContainerInstances).AnyTimes()
	ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.RunTask).AnyTimes()
	ecsMock.EXPECT().StopTask(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.StopTask).AnyTimes()

	albMock := mock_awsiface.NewMockAlbClient(ctrl)
	albMock.EXPECT().DescribeTargetGroups(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetGroups).AnyTimes()
	albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetHealth).AnyTimes()
	albMock.EXPECT().DescribeTargetGroupAttributes(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTargetGroupAttibutes).AnyTimes()
	albMock.EXPECT().RegisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTarget).AnyTimes()
	albMock.EXPECT().DeregisterTargets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeregisterTarget).AnyTimes()

	ec2Mock := mock_awsiface.NewMockEc2Client(ctrl)
	ec2Mock.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeSubnets).AnyTimes()
	ec2Mock.EXPECT().DescribeInstances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeInstances).AnyTimes()
	td, _ := mocker.RegisterTaskDefinition(context.Background(), envars.TaskDefinitionInput)
	if currentTaskCount >= 0 {
		input := *envars.ServiceDefinitionInput
		input.TaskDefinition = td.TaskDefinition.TaskDefinitionArn
		input.DesiredCount = aws.Int32(int32(currentTaskCount))
		input.LaunchType = launchType
		svc, _ := mocker.CreateService(context.Background(), &input)
		if len(svc.Service.LoadBalancers) > 0 {
			_, _ = mocker.RegisterTarget(context.Background(), &elbv2.RegisterTargetsInput{
				TargetGroupArn: svc.Service.LoadBalancers[0].TargetGroupArn,
			})
		}
	}
	return mocker, ecsMock, albMock, ec2Mock
}

func DefaultEnvars() *env.Envars {
	d, _ := os.ReadFile("fixtures/task-definition.json")
	var taskDefinition ecs.RegisterTaskDefinitionInput
	if err := json.Unmarshal(d, &taskDefinition); err != nil {
		log.Fatalf(err.Error())
	}
	return &env.Envars{
		Region:                 "us-west-2",
		Cluster:                "cage-test",
		Service:                "service",
		ServiceDefinitionInput: ReadServiceDefinition("fixtures/service.json"),
		TaskDefinitionInput:    &taskDefinition,
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
