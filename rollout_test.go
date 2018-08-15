package main

import (
	"github.com/aws/aws-sdk-go/service/ecs"
	"testing"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mock/mock_ecs"
	"github.com/loilo-inc/canarycage/mock/mock_cloudwatch"
	"github.com/apex/log"
	"github.com/loilo-inc/canarycage/test"
	"github.com/aws/aws-sdk-go/aws"
	"io/ioutil"
	"encoding/base64"
	"encoding/json"
)

const kCurrentServiceName = "service-current"
const kNextServiceName = "service-next"

func DefaultEnvars() *Envars {
	d, _ := ioutil.ReadFile("fixtures/task-definition.json")
	o := base64.StdEncoding.EncodeToString(d)
	return &Envars{
		Region:                   aws.String("us-west-2"),
		RollOutPeriod:            aws.Int64(0),
		LoadBalancerArn:          aws.String("hoge/app/1111/hoge"),
		Cluster:                  aws.String("cage-test"),
		CurrentServiceName:       aws.String(kCurrentServiceName),
		NextServiceName:          aws.String(kNextServiceName),
		NextTaskDefinitionBase64: aws.String(o),
		AvailabilityThreshold:    aws.Float64(0.9970),
		ResponseTimeThreshold:    aws.Float64(1),
		UpdateServicePeriod:      aws.Int64(0),
		UpdateServiceTimeout:     aws.Int64(1),
	}
}

func TestStartGradualRollOut(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	arr := []int64{1, 2, 15}
	for _, v := range arr {
		log.Debugf("====")
		envars := DefaultEnvars()
		ctrl := gomock.NewController(t)
		ctx, ecsMock, cwMock := envars.Setup(ctrl, v)
		if ctx.ServiceSize() != 1 {
			t.Fatalf("current service not setup")
		}
		if taskCnt := ctx.TaskSize(); taskCnt != int(v) {
			t.Fatalf("current tasks not setup: %d/%d", v, taskCnt)
		}
		err := envars.StartGradualRollOut(ecsMock, cwMock)
		if err != nil {
			t.Fatalf("%s", err)
		}
	}
}

func (envars *Envars) Setup(ctrl *gomock.Controller, currentTaskCount int64) (*test.MockContext, *mock_ecs.MockECSAPI, *mock_cloudwatch.MockCloudWatchAPI) {
	ecsMock := mock_ecs.NewMockECSAPI(ctrl)
	cwMock := mock_cloudwatch.NewMockCloudWatchAPI(ctrl)
	ctx := test.NewMockContext()
	ecsMock.EXPECT().CreateService(gomock.Any()).DoAndReturn(ctx.CreateService).AnyTimes()
	ecsMock.EXPECT().UpdateService(gomock.Any()).DoAndReturn(ctx.UpdateService).AnyTimes()
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
	o, _ := base64.StdEncoding.DecodeString(*envars.NextTaskDefinitionBase64)
	var register *ecs.RegisterTaskDefinitionInput
	_ = json.Unmarshal(o, register)
	td, _ := ctx.RegisterTaskDefinition(register)
	a := &ecs.CreateServiceInput{
		ServiceName: envars.CurrentServiceName,
		LoadBalancers: []*ecs.LoadBalancer{
			{
				TargetGroupArn: aws.String("arn://tg"),
				ContainerName:  aws.String("container"),
				ContainerPort:  aws.Int64(80),
			},
		},
		TaskDefinition: td.TaskDefinition.TaskDefinitionArn,
		DesiredCount:   aws.Int64(currentTaskCount),
	}
	_, _ = ctx.CreateService(a)
	return ctx, ecsMock, cwMock
}

func TestEnvars_Rollback(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	envars := DefaultEnvars()
	ctrl := gomock.NewController(t)
	ctx, e, _ := envars.Setup(ctrl, 2)
	ctx.CreateService(&ecs.CreateServiceInput{
		Cluster:        envars.Cluster,
		ServiceName:    envars.NextServiceName,
		TaskDefinition: aws.String("arn://current"),
		DesiredCount:   aws.Int64(12),
	})
	log.Debugf("%d", ctx.ServiceSize())
	err := envars.Rollback(e, aws.Int64(10))
	if err != nil {
		t.Fatal(err.Error())
	}
	if ctx.ServiceSize() != 1 {
		t.Fatal("next service still exists")
	}
	o, _ := e.ListTasks(&ecs.ListTasksInput{
		ServiceName: envars.CurrentServiceName,
	})
	if l := len(o.TaskArns); l != 10 {
		t.Fatalf("next service was not rollbacked: E: %d, A: %d", 10, l)
	}
}
