package main

import (
	"github.com/aws/aws-sdk-go/service/ecs"
	"testing"
	"time"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mock/mock_ecs"
	"github.com/loilo-inc/canarycage/mock/mock_cloudwatch"
	"fmt"
	"github.com/apex/log"
	"github.com/loilo-inc/canarycage/test"
	"github.com/aws/aws-sdk-go/aws"
)

const kCurrentServiceName = "service-current"

func DefaultEnvars() *Envars {
	return &Envars{
		Region:                   "us-west-2",
		ReleaseStage:             "test",
		RollOutPeriod:            time.Duration(0) * time.Second,
		LoadBalancerArn:          "hoge/app/1111/hoge",
		Cluster:                  "cage-test",
		ServiceName:              kCurrentServiceName,
		CurrentTaskDefinitionArn: "current-task-definition",
		NextTaskDefinitionArn:    "next-task-definition",
		AvailabilityThreshold:    0.9970,
		ResponseTimeThreshold:    1,
	}
}

func TestStartGradualRollOut(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	arr := [][]int{{2}, {2}, {2}, {15}}
	for _, v := range arr {
		envars := DefaultEnvars()
		current := v[0]
		ctrl := gomock.NewController(t)
		ctx, ecsMock, cwMock := envars.Setup(ctrl, current)
		if ctx.ServiceSize() != 1 {
			t.Fatalf("current service not setup")
		}
		if taskCnt := ctx.TaskSize(); taskCnt != v[0] {
			t.Fatalf("current tasks not setup: %d", taskCnt)
		}
		err := envars.StartGradualRollOut(ecsMock, cwMock)
		if err != nil {
			t.Fatalf("%s", err)
		}
	}
}

func (envars *Envars) Setup(ctrl *gomock.Controller, currentTaskCount int) (*test.MockContext, *mock_ecs.MockECSAPI, *mock_cloudwatch.MockCloudWatchAPI) {
	ecsMock := mock_ecs.NewMockECSAPI(ctrl)
	cwMock := mock_cloudwatch.NewMockCloudWatchAPI(ctrl)
	ctx := test.NewMockContext()
	ecsMock.EXPECT().CreateService(gomock.Any()).DoAndReturn(ctx.CreateService).AnyTimes()
	ecsMock.EXPECT().DeleteService(gomock.Any()).DoAndReturn(ctx.DeleteService).AnyTimes()
	ecsMock.EXPECT().StartTask(gomock.Any()).DoAndReturn(ctx.StartTask).AnyTimes()
	ecsMock.EXPECT().StopTask(gomock.Any()).DoAndReturn(ctx.StopTask).AnyTimes()
	ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any()).DoAndReturn(ctx.RegisterTaskDefinition).AnyTimes()
	ecsMock.EXPECT().WaitUntilServicesStable(gomock.Any()).DoAndReturn(ctx.WaitUntilServicesStable).AnyTimes()
	ecsMock.EXPECT().WaitUntilServicesInactive(gomock.Any()).DoAndReturn(ctx.WaitUntilServicesInactive).AnyTimes()
	ecsMock.EXPECT().DescribeServices(gomock.Any()).DoAndReturn(ctx.DescribeServices).AnyTimes()
	ecsMock.EXPECT().DescribeTasks(gomock.Any()).DoAndReturn(ctx.DescribeTasks).AnyTimes()
	ecsMock.EXPECT().WaitUntilTasksRunning(gomock.Any()).DoAndReturn(ctx.WaitUntilTasksRunning).AnyTimes()
	ecsMock.EXPECT().WaitUntilTasksStopped(gomock.Any()).DoAndReturn(ctx.WaitUntilTasksStopped).AnyTimes()
	ecsMock.EXPECT().ListTasks(gomock.Any()).DoAndReturn(ctx.ListTasks).AnyTimes()
	cwMock.EXPECT().GetMetricStatistics(gomock.Any()).DoAndReturn(ctx.GetMetricStatics).AnyTimes()
	a := &ecs.CreateServiceInput{
		ServiceName: &envars.ServiceName,
		LoadBalancers: []*ecs.LoadBalancer{
			{
				TargetGroupArn: aws.String("arn://tg"),
				ContainerName:  aws.String("container"),
				ContainerPort:  aws.Int64(80),
			},
		},
		TaskDefinition: &envars.CurrentTaskDefinitionArn,
		DesiredCount:   aws.Int64(2),
	}
	o, _ := ctx.CreateService(a)
	group := fmt.Sprintf("service:%s", envars.ServiceName)
	for i := int(*o.Service.RunningCount); i < currentTaskCount; i++ {
		_, err := ctx.StartTask(&ecs.StartTaskInput{
			Cluster:        &envars.Cluster,
			Group:          &group,
			TaskDefinition: &envars.CurrentTaskDefinitionArn,
		})
		if err != nil {
			log.Fatal(err.Error())
		}
	}
	return ctx, ecsMock, cwMock
}

func TestEnvars_Rollback(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	envars := DefaultEnvars()
	ctrl := gomock.NewController(t)
	ctx, e, _ := envars.Setup(ctrl, 2)
	group := fmt.Sprintf("service:%s", envars.ServiceName)
	for i := 0; i < 12; i++ {
		_, err := ctx.StartTask(&ecs.StartTaskInput{
			Cluster:        &envars.Cluster,
			Group:          &group,
			TaskDefinition: &envars.NextTaskDefinitionArn,
		})
		if err != nil {
			log.Fatal(err.Error())
		}
	}
	currentService, _ := ctx.GetService(kCurrentServiceName)
	log.Debugf("%d", ctx.ServiceSize())
	err := envars.Rollback(e, 10)
	if err != nil {
		t.Fatal(err.Error())
	}
	if ctx.ServiceSize() != 1 {
		t.Fatal("next service still exists")
	}
	o, err := e.ListTasks(&ecs.ListTasksInput{
		ServiceName: currentService.ServiceName,
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if l := len(o.TaskArns); l != 10 {
		t.Fatalf("next service was not rollbacked: E: %d, A: %d", 10, l)
	}
}
