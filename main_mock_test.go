package main

import (
	"github.com/aws/aws-sdk-go/service/ecs"
	"testing"
	"io/ioutil"
	"time"
	"encoding/base64"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mock/mock_ecs"
	"github.com/loilo-inc/canarycage/mock/mock_cloudwatch"
	"fmt"
	"github.com/apex/log"
	"github.com/loilo-inc/canarycage/test"
)

const kCurrentServiceName = "service-current"

func DefaultEnvars() *Envars {
	serviceJson, _ := ioutil.ReadFile("fixtures/service-definition-next.json")
	taskJson, _ := ioutil.ReadFile("fixtures/task-definition-next.json")
	return &Envars{
		Region:                      "us-west-2",
		ReleaseStage:                "test",
		RollOutPeriod:               time.Duration(0) * time.Second,
		LoadBalancerArn:             "hoge/app/1111/hoge",
		Cluster:                     "cage-test",
		CurrentServiceName:          kCurrentServiceName,
		CurrentTaskDefinitionArn:    "current-task-definition",
		NextTaskDefinitionBase64:    base64.StdEncoding.EncodeToString([]byte(taskJson)),
		NextServiceDefinitionBase64: base64.StdEncoding.EncodeToString([]byte(serviceJson)),
		AvailabilityThreshold:       0.9970,
		ResponseTimeThreshold:       1,
	}
}

func TestStartGradualRollOut(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	arr := [][]int{{2, 1}, {2, 2}, {2, 15}, {15, 2}}
	for _, v := range arr {
		envars := DefaultEnvars()
		current := v[0]
		next := v[1]
		expect := v[0]
		ctrl := gomock.NewController(t)
		ctx, ecsMock, cwMock := envars.Setup(ctrl, current, next)
		if ctx.ServiceSize() != 1 {
			t.Fatalf("current service not setup")
		}
		if taskCnt := ctx.TaskSize(); taskCnt != expect {
			t.Fatalf("current tasks not setup: %d / %d", taskCnt, expect)
		}
		err := envars.StartGradualRollOut(ecsMock, cwMock)
		if err != nil {
			t.Fatalf("%s", err)
		}
	}
}

func (envars *Envars) Setup(ctrl *gomock.Controller, currentTaskCount int, nextStartTaskCount int) (*test.MockContext, *mock_ecs.MockECSAPI, *mock_cloudwatch.MockCloudWatchAPI) {
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
	ecsMock.EXPECT().WaitUntilTasksRunning(gomock.Any()).DoAndReturn(ctx.WaitUntilTasksRunning).AnyTimes()
	ecsMock.EXPECT().WaitUntilTasksStopped(gomock.Any()).DoAndReturn(ctx.WaitUntilTasksStopped).AnyTimes()
	ecsMock.EXPECT().ListTasks(gomock.Any()).DoAndReturn(ctx.ListTasks).AnyTimes()
	cwMock.EXPECT().GetMetricStatistics(gomock.Any()).DoAndReturn(ctx.GetMetricStatics).AnyTimes()
	taskdef, _ := ioutil.ReadFile("fixtures/task-definition-current.json")
	servicedef, _ := ioutil.ReadFile("fixtures/service-definition-current.json")
	t, _ := UnmarshalTaskDefinition(base64.StdEncoding.EncodeToString(taskdef))
	s, _ := UnmarshalServiceDefinition(base64.StdEncoding.EncodeToString(servicedef))
	taskDefinition, _ := ctx.RegisterTaskDefinition(t)
	o, _ := ctx.CreateService(s)
	for i := int(*o.Service.RunningCount); i < currentTaskCount; i++ {
		group := fmt.Sprintf("service:%s", *o.Service.ServiceName)
		_, err := ctx.StartTask(&ecs.StartTaskInput{
			Cluster:        &envars.Cluster,
			Group:          &group,
			TaskDefinition: taskDefinition.TaskDefinition.TaskDefinitionArn,
		})
		if err != nil {
			log.Error(err.Error())
		}
	}
	return ctx, ecsMock, cwMock
}

//func TestEnvars_RollOut(t *testing.T) {
//	envars := DefaultEnvars()
//	ctrl := gomock.NewController(t)
//	ctx, e, _ := envars.Setup(ctrl, 10, 0)
//	currentService, _ := ctx.GetService(kCurrentServiceName)
//	nt, _ := envars.CreateNextTaskDefinition(e)
//	nsvr, _ := envars.CreateNextService(e, nt.TaskDefinitionArn)
//	o, err := envars.RollOut(e, currentService, nsvr, 10, 0, 2)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	if len(o) != 2 {
//		t.Fatalf("E: %d, A: %d", 2, len(o))
//	}
//}

func TestEnvars_Rollback(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	envars := DefaultEnvars()
	ctrl := gomock.NewController(t)
	ctx, e, _ := envars.Setup(ctrl, 2, 12)
	currentService, _ := ctx.GetService(kCurrentServiceName)
	nt, _ := envars.CreateNextTaskDefinition(e)
	nextService, _ := envars.CreateNextService(e, nt.TaskDefinitionArn)
	log.Debugf("%d", ctx.ServiceSize())
	err := envars.Rollback(e, *nextService.ServiceName, 10)
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
