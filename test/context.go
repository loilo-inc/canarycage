package test

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/google/uuid"
	"regexp"
	"fmt"
	"sync"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/apex/log"
)

type MockContext struct {
	services map[string]*ecs.Service
	tasks    map[string]*ecs.Task
	mux      sync.Mutex
}

func NewMockContext() *MockContext {
	return &MockContext{
		services: make(map[string]*ecs.Service),
		tasks:    make(map[string]*ecs.Task),
	}
}

func (ctx *MockContext) GetTask(id string) (*ecs.Task, bool) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	o, ok := ctx.tasks[id]
	return o, ok
}

func (ctx *MockContext) TaskSize() int {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	return len(ctx.tasks)
}

func (ctx *MockContext) GetService(id string) (*ecs.Service, bool) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	o, ok := ctx.services[id]
	return o, ok
}

func (ctx *MockContext) ServiceSize() int {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	return len(ctx.services)
}

func (ctx *MockContext) GetMetricStatics(input *cloudwatch.GetMetricStatisticsInput) (*cloudwatch.GetMetricStatisticsOutput, error) {
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

func (ctx *MockContext) CreateService(input *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
	idstr := uuid.New().String()
	lt := "FARGATE"
	st := "ACTIVE"
	ret := &ecs.Service{
		ServiceName:   input.ServiceName,
		RunningCount:  aws.Int64(0),
		LaunchType:    &lt,
		LoadBalancers: input.LoadBalancers,
		DesiredCount:  input.DesiredCount,
		Status:        &st,
		ServiceArn:    &idstr,
	}
	ctx.mux.Lock()
	ctx.services[*input.ServiceName] = ret
	ctx.mux.Unlock()
	log.Debugf("%s: %d, desired=%d", *input.ServiceName, *ret.RunningCount, *input.DesiredCount)
	for i := 0; i < int(*input.DesiredCount); i++ {
		ctx.StartTask(&ecs.StartTaskInput{
			Cluster:        input.Cluster,
			Group:          aws.String(fmt.Sprintf("service:%s", *input.ServiceName)),
			TaskDefinition: input.TaskDefinition,
		})
	}
	log.Debugf("%s: %d", *input.ServiceName, *ret.RunningCount)
	return &ecs.CreateServiceOutput{
		Service: ret,
	}, nil
}
func (ctx *MockContext) DeleteService(input *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	service := ctx.services[*input.Service]
	delete(ctx.services, *input.Service)
	return &ecs.DeleteServiceOutput{
		Service: service,
	}, nil
}

func (ctx *MockContext) RegisterTaskDefinition(input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	idstr := uuid.New().String()
	return &ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: &idstr,
		},
	}, nil
}

func (ctx *MockContext) StartTask(input *ecs.StartTaskInput) (*ecs.StartTaskOutput, error) {
	regex := regexp.MustCompile("service:(.+?)$")
	m := regex.FindStringSubmatch(*input.Group)
	id := uuid.New()
	idstr := id.String()
	ret := &ecs.Task{
		TaskArn:           &idstr,
		ClusterArn:        input.Cluster,
		TaskDefinitionArn: input.TaskDefinition,
		Group:             input.Group,
	}
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	ctx.tasks[idstr] = ret
	s := ctx.services[m[1]]
	*s.RunningCount += 1
	return &ecs.StartTaskOutput{
		Tasks: []*ecs.Task{ret},
	}, nil
}

func (ctx *MockContext) StopTask(input *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	ret := ctx.tasks[*input.Task]
	delete(ctx.tasks, *input.Task)
	reg := regexp.MustCompile("service:(.+?)$")
	m := reg.FindStringSubmatch(*ret.Group)
	service := ctx.services[m[1]]
	*service.RunningCount -= 1
	return &ecs.StopTaskOutput{
		Task: ret,
	}, nil
}

func (ctx *MockContext) ListTasks(input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	var ret []*string
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range ctx.tasks {
		group := fmt.Sprintf("service:%s", *input.ServiceName)
		if *v.Group == group {
			ret = append(ret, v.TaskArn)
		}
	}
	return &ecs.ListTasksOutput{
		TaskArns: ret,
	}, nil
}

func (ctx *MockContext) WaitUntilServicesStable(input *ecs.DescribeServicesInput) (error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range input.Services {
		if _, ok := ctx.services[*v]; !ok {
			return errors.New(fmt.Sprintf("service:%s not found", *v))
		}
	}
	return nil
}

func (ctx *MockContext) DescribeServices(input *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	var ret []*ecs.Service
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range input.Services {
		if s, ok := ctx.services[*v]; ok {
			ret = append(ret, s)
		}
	}
	return &ecs.DescribeServicesOutput{
		Services: ret,
	}, nil
}

func (ctx *MockContext) WaitUntilServicesInactive(input *ecs.DescribeServicesInput) (error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range input.Services {
		if _, ok := ctx.services[*v]; ok {
			return errors.New(fmt.Sprintf("service:%s found", *v))
		}
	}
	return nil
}

func (ctx *MockContext) WaitUntilTasksRunning(input *ecs.DescribeTasksInput) (error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range input.Tasks {
		if _, ok := ctx.tasks[*v]; !ok {
			return errors.New(fmt.Sprintf("task:%s not running", *v))
		}
	}
	return nil
}
func (ctx *MockContext) WaitUntilTasksStopped(input *ecs.DescribeTasksInput) (error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range input.Tasks {
		if _, ok := ctx.tasks[*v]; ok {
			return errors.New(fmt.Sprintf("task:%s found", *v))
		}
	}
	return nil
}
