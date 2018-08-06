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
	"regexp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/apex/log"
)

const kCurrentServiceName = "current-service"
const kNextServiceName = "next-service"

func TestStartGradualRollOut(t *testing.T) {
	serviceJson, _ := ioutil.ReadFile("fixtures/service-definition.json")
	taskJson, _ := ioutil.ReadFile("fixtures/task-definition.json")
	envars := Envars{
		Region:                      "us-west-2",
		ReleaseStage:                "test",
		RollOutPeriod:               time.Duration(5) * time.Second,
		LoadBalancerArn:             "hoge/app/1111/hoge",
		Cluster:                     "cage-test",
		CurrentServiceArn:           "current-service",
		CurrentTaskDefinitionArn:    "current-task-definition",
		NextTaskDefinitionBase64:    base64.StdEncoding.EncodeToString([]byte(serviceJson)),
		NextServiceDefinitionBase64: base64.StdEncoding.EncodeToString([]byte(taskJson)),
		NextServiceName:             kNextServiceName,
		AvailabilityThreshold:       0.9970,
		ResponseTimeThreshold:       1,
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ecsMock := mock_ecs.NewMockECSAPI(ctrl)
	ctx := IntegrationContext{
		tasks:    make(map[string]ecs.Task),
		services: make(map[string]ecs.Service),
		envars:   envars,
	}
	log.SetLevel(log.DebugLevel)
	ecsMock.EXPECT().CreateService(gomock.Any()).DoAndReturn(ctx.CreateService)
	ecsMock.EXPECT().DeleteService(gomock.Any()).DoAndReturn(ctx.DeleteService)
	ecsMock.EXPECT().StartTask(gomock.Any()).DoAndReturn(ctx.StartTask)
	ecsMock.EXPECT().StopTask(gomock.Any()).DoAndReturn(ctx.StopTask)
	ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any()).DoAndReturn(ctx.RegisterTaskDefinition)
	ecsMock.EXPECT().WaitUntilServicesStable(gomock.Any()).DoAndReturn(ctx.WaitUntilServicesStable)
	ecsMock.EXPECT().WaitUntilServicesInactive(gomock.Any()).DoAndReturn(ctx.WaitUntilServicesInactive)
	ecsMock.EXPECT().DescribeServices(gomock.Any()).DoAndReturn(ctx.DescribeServices)
	ecsMock.EXPECT().WaitUntilTasksRunning(gomock.Any()).DoAndReturn(ctx.WaitUntilTasksRunning)
	ecsMock.EXPECT().WaitUntilTasksStopped(gomock.Any()).DoAndReturn(ctx.WaitUntilTasksStopped)
	ecsMock.EXPECT().ListTasks(gomock.Any()).DoAndReturn(ctx.ListTasks)
	cwMock := mock_cloudwatch.NewMockCloudWatchAPI(ctrl)
	cwMock.EXPECT().GetMetricStatistics(gomock.Any()).DoAndReturn(ctx.GetMetricStatics)
	envars.StartGradualRollOut(ecsMock, cwMock)
}

type IntegrationContext struct {
	services map[string]ecs.Service
	tasks    map[string]ecs.Task
	envars   Envars
}

func (ctx *IntegrationContext) GetMetricStatics(input *cloudwatch.GetMetricStatisticsInput) (*cloudwatch.GetMetricStatisticsOutput, error) {
	var ret cloudwatch.Datapoint
	switch *input.MetricName {
	case "RequestCount":
		sum := 100000.0
		ret.Sum = &sum
	case "HTTPCode_ELB_5XX_Count":
		sum := 1.0
		ret.Sum = &sum
	case "HTTPCode_Target_5XX_Count":
		sum := 1.0
		ret.Sum = &sum
	case "TargetResponseTime":
		average := 0.1
		ret.Average = &average
	}
	return &cloudwatch.GetMetricStatisticsOutput{
		Datapoints: []*cloudwatch.Datapoint{
			&ret,
		},
	}, nil
}

func (ctx *IntegrationContext) CreateService(input *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
	idstr := uuid.New().String()
	lt := "FARGATE"
	st := "ACTIVE"
	ret := ecs.Service{
		ServiceName:   input.ServiceName,
		RunningCount:  input.DesiredCount,
		LaunchType:    &lt,
		LoadBalancers: input.LoadBalancers,
		Status:        &idstr,
		ServiceArn:    &st,
	}
	ctx.services[idstr] = ret
	return &ecs.CreateServiceOutput{
		Service: &ret,
	}, nil
}
func (ctx *IntegrationContext) DeleteService(input *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error) {
	service := ctx.services[*input.Service]
	delete(ctx.services, *input.Service)
	return &ecs.DeleteServiceOutput{
		Service: &service,
	}, nil
}

func (ctx *IntegrationContext) RegisterTaskDefinition(input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	idstr := uuid.New().String()
	return &ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: &idstr,
		},
	}, nil
}

func (ctx *IntegrationContext) StartTask(input *ecs.StartTaskInput) (*ecs.StartTaskOutput, error) {
	regex := regexp.MustCompile("service:(.+?)$")
	m := regex.FindStringSubmatch(*input.Group)
	s := ctx.services[m[1]]
	*s.RunningCount += 1
	id := uuid.New()
	idstr := id.String()
	ret := ecs.Task{
		TaskArn:           &idstr,
		ClusterArn:        input.Cluster,
		TaskDefinitionArn: input.TaskDefinition,
		Group:             input.Group,
	}
	ctx.tasks[idstr] = ret
	return &ecs.StartTaskOutput{
		Tasks: []*ecs.Task{ &ret },
	}, nil
}

func (ctx *IntegrationContext) StopTask(input *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	ret := ctx.tasks[*input.Task]
	delete(ctx.tasks, *input.Task)
	return &ecs.StopTaskOutput{
		Task: &ret,
	}, nil
}

func (ctx *IntegrationContext) ListTasks(input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	var ret []*string
	for _, v := range ctx.tasks {
		ret = append(ret, v.TaskArn)
	}
	return &ecs.ListTasksOutput{
		TaskArns: ret,
	}, nil
}

func (ctx *IntegrationContext) WaitUntilServicesStable(input *ecs.DescribeServicesInput) (error) {
	for _, v := range input.Services {
		if _, ok := ctx.services[*v]; !ok {
			return errors.New(fmt.Sprintf("service:%s not found", *v))
		}
	}
	return nil
}

func (ctx *IntegrationContext) DescribeServices(input *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	var ret []*ecs.Service
	for _, v := range ctx.services {
		ret = append(ret, &v)
	}
	return &ecs.DescribeServicesOutput{
		Services: ret,
	}, nil
}

func (ctx *IntegrationContext) WaitUntilServicesInactive(input *ecs.DescribeServicesInput) (error) {
	for _, v := range input.Services {
		if _, ok := ctx.services[*v]; ok {
			return errors.New(fmt.Sprintf("service:%s found", *v))
		}
	}
	return nil
}

func (ctx *IntegrationContext) WaitUntilTasksRunning(input *ecs.DescribeTasksInput) (error) {
	for _, v := range input.Tasks {
		if _, ok := ctx.tasks[*v]; !ok {
			return errors.New(fmt.Sprintf("task:%s not running", *v))
		}
	}
	return nil
}
func (ctx *IntegrationContext) WaitUntilTasksStopped(input *ecs.DescribeTasksInput) (error) {
	for _, v := range input.Tasks {
		if _, ok := ctx.tasks[*v]; ok {
			return errors.New(fmt.Sprintf("task:%s found", *v))
		}
	}
	return nil
}
