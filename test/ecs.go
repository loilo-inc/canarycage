package test

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
)

type EcsServer struct {
	*commons
	ipv4 int
}

func (ctx *EcsServer) CreateService(c context.Context, input *ecs.CreateServiceInput, _ ...func(options *ecs.Options)) (*ecs.CreateServiceOutput, error) {
	idstr := uuid.New().String()
	st := "ACTIVE"
	if old, ok := ctx.Services[*input.ServiceName]; ok {
		if *old.Status == "ACTIVE" {
			return nil, fmt.Errorf("service already exists: %s", *input.ServiceName)
		}
	}
	ret := &types.Service{
		ServiceName:                   input.ServiceName,
		RunningCount:                  0,
		LaunchType:                    input.LaunchType,
		LoadBalancers:                 input.LoadBalancers,
		DesiredCount:                  *input.DesiredCount,
		TaskDefinition:                input.TaskDefinition,
		HealthCheckGracePeriodSeconds: aws.Int32(0),
		Status:                        &st,
		ServiceArn:                    &idstr,
		PlatformVersion:               input.PlatformVersion,
		ServiceRegistries:             input.ServiceRegistries,
		NetworkConfiguration:          input.NetworkConfiguration,
		Deployments: []types.Deployment{
			{
				DesiredCount:   *input.DesiredCount,
				LaunchType:     input.LaunchType,
				RunningCount:   *input.DesiredCount,
				Status:         &st,
				TaskDefinition: input.TaskDefinition,
			},
		},
	}
	ctx.mux.Lock()
	ctx.Services[*input.ServiceName] = ret
	ctx.mux.Unlock()
	log.Debugf("%s: running=%d, desired=%d", *input.ServiceName, ret.RunningCount, *input.DesiredCount)
	for i := 0; i < int(*input.DesiredCount); i++ {
		if _, err := ctx.StartTask(c, &ecs.StartTaskInput{
			Cluster:              input.Cluster,
			Group:                aws.String(fmt.Sprintf("service:%s", *input.ServiceName)),
			NetworkConfiguration: input.NetworkConfiguration,
			TaskDefinition:       input.TaskDefinition,
		}); err != nil {
			log.Fatalf("failed to start task: %v", err)
		}
	}
	ctx.mux.Lock()
	ctx.Services[*input.ServiceName].RunningCount = *input.DesiredCount
	ctx.mux.Unlock()
	log.Debugf("%s: running=%d", *input.ServiceName, ret.RunningCount)
	return &ecs.CreateServiceOutput{
		Service: ret,
	}, nil
}

func (ctx *EcsServer) UpdateService(c context.Context, input *ecs.UpdateServiceInput, _ ...func(options *ecs.Options)) (*ecs.UpdateServiceOutput, error) {
	ctx.mux.Lock()
	s := ctx.Services[*input.Service]
	ctx.mux.Unlock()
	nextDesiredCount := s.DesiredCount
	nextTaskDefinition := s.TaskDefinition
	if input.TaskDefinition != nil {
		nextTaskDefinition = input.TaskDefinition
	}
	if input.DesiredCount != nil {
		nextDesiredCount = *input.DesiredCount
	}
	if diff := nextDesiredCount - s.DesiredCount; diff > 0 {
		log.Debugf("diff=%d", diff)
		// scale
		for i := 0; i < int(diff); i++ {
			ctx.StartTask(c, &ecs.StartTaskInput{
				Cluster:        input.Cluster,
				Group:          aws.String(fmt.Sprintf("service:%s", *input.Service)),
				TaskDefinition: nextTaskDefinition,
			})
		}
	} else if diff < 0 {
		// descale
		var i int32 = 0
		max := -diff
		for k, v := range ctx.Tasks {
			reg := regexp.MustCompile("service:" + *s.ServiceName)
			if reg.MatchString(*v.Group) {
				ctx.StopTask(c, &ecs.StopTaskInput{
					Cluster: input.Cluster,
					Task:    &k,
				})
				i++
				if i >= max {
					break
				}
			}
		}
	}
	ctx.mux.Lock()
	s.DesiredCount = nextDesiredCount
	s.TaskDefinition = nextTaskDefinition
	s.RunningCount = nextDesiredCount
	s.PlatformVersion = input.PlatformVersion
	s.ServiceRegistries = input.ServiceRegistries
	s.NetworkConfiguration = input.NetworkConfiguration
	s.LoadBalancers = input.LoadBalancers
	s.Deployments = []types.Deployment{
		{
			DesiredCount:   nextDesiredCount,
			LaunchType:     s.LaunchType,
			RunningCount:   nextDesiredCount,
			Status:         s.Status,
			TaskDefinition: s.TaskDefinition,
		},
	}
	ctx.mux.Unlock()
	return &ecs.UpdateServiceOutput{
		Service: s,
	}, nil
}

func (ctx *EcsServer) DeleteService(c context.Context, input *ecs.DeleteServiceInput, _ ...func(options *ecs.Options)) (*ecs.DeleteServiceOutput, error) {
	service := ctx.Services[*input.Service]
	reg := regexp.MustCompile(fmt.Sprintf("service:%s", *service.ServiceName))
	for _, v := range ctx.Tasks {
		if reg.MatchString(*v.Group) {
			_, err := ctx.StopTask(c, &ecs.StopTaskInput{
				Cluster: input.Cluster,
				Task:    v.TaskArn,
			})
			if err != nil {
				return nil, err
			}
		}
	}
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	service.Status = aws.String("INACTIVE")
	return &ecs.DeleteServiceOutput{Service: service}, nil
}

func (ctx *EcsServer) RegisterTaskDefinition(_ context.Context, input *ecs.RegisterTaskDefinitionInput, _ ...func(options *ecs.Options)) (*ecs.RegisterTaskDefinitionOutput, error) {
	td, err := ctx.TaskDefinitions.Register(input)
	if err != nil {
		return nil, err
	}
	return &ecs.RegisterTaskDefinitionOutput{TaskDefinition: td}, nil
}

func (ctx *EcsServer) StartTask(_ context.Context, input *ecs.StartTaskInput, _ ...func(options *ecs.Options)) (*ecs.StartTaskOutput, error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	td := ctx.TaskDefinitions.Get(*input.TaskDefinition)
	if td == nil {
		return nil, fmt.Errorf("task definition not found: %s", *input.TaskDefinition)
	}
	taskArn := fmt.Sprintf("arn:aws:ecs:us-west-2:012345678910:task/%s", uuid.New().String())
	ctx.ipv4++
	attachment := types.Attachment{
		Status: aws.String("ATTACHED"),
		Type:   aws.String("ElasticNetworkInterface"),
		Details: []types.KeyValuePair{
			{
				Name:  aws.String("privateIPv4Address"),
				Value: aws.String(fmt.Sprintf("127.0.0.%d", ctx.ipv4)),
			},
		},
	}
	if input.NetworkConfiguration != nil {
		subnet := input.NetworkConfiguration.AwsvpcConfiguration.Subnets[0]
		attachment.Details = append(attachment.Details, types.KeyValuePair{
			Name:  aws.String("subnetId"),
			Value: &subnet,
		})
	}
	containers := make([]types.Container, len(td.ContainerDefinitions))
	for i, v := range td.ContainerDefinitions {
		containers[i] = types.Container{
			Name:       v.Name,
			Image:      v.Image,
			LastStatus: aws.String("RUNNING"),
		}
		if v.HealthCheck != nil {
			containers[i].HealthStatus = "HEALTHY"
		} else {
			containers[i].HealthStatus = "UNKNOWN"
		}
	}

	ret := types.Task{
		TaskArn:           &taskArn,
		ClusterArn:        input.Cluster,
		TaskDefinitionArn: input.TaskDefinition,
		Group:             input.Group,
		Containers:        containers,
	}
	ctx.Tasks[taskArn] = &ret
	var launchType types.LaunchType
	if len(input.ContainerInstances) > 0 {
		launchType = types.LaunchTypeEc2
	} else {
		launchType = types.LaunchTypeFargate
	}
	ret.LaunchType = launchType
	if launchType == types.LaunchTypeFargate {
		ret.Attachments = []types.Attachment{attachment}
	} else {
		ret.ContainerInstanceArn = aws.String("arn:aws:ecs:us-west-2:1234567890:container-instance/12345678-hoge-hoge-1234-1f2o3o4ba5r")
	}
	ret.LastStatus = aws.String("RUNNING")
	return &ecs.StartTaskOutput{
		Tasks: []types.Task{ret},
	}, nil
}

func (ctx *EcsServer) RunTask(c context.Context, input *ecs.RunTaskInput, _ ...func(options *ecs.Options)) (*ecs.RunTaskOutput, error) {
	o, err := ctx.StartTask(c, &ecs.StartTaskInput{
		Cluster:              input.Cluster,
		Group:                input.Group,
		TaskDefinition:       input.TaskDefinition,
		NetworkConfiguration: input.NetworkConfiguration,
	})
	if err != nil {
		return nil, err
	}
	return &ecs.RunTaskOutput{
		Tasks: o.Tasks,
	}, nil
}

func (ctx *EcsServer) StopTask(_ context.Context, input *ecs.StopTaskInput, _ ...func(options *ecs.Options)) (*ecs.StopTaskOutput, error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	log.Debugf("stop: %s", *input.Task)
	ret, ok := ctx.Tasks[*input.Task]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", *input.Task)
	}
	for i := range ret.Containers {
		v := &ret.Containers[i]
		v.ExitCode = aws.Int32(0)
		v.LastStatus = aws.String("STOPPED")
	}
	ret.LastStatus = aws.String("STOPPED")
	ret.DesiredStatus = aws.String("STOPPED")
	service, ok := ctx.Services[*ret.Group]
	if ok {
		service.RunningCount -= 1
	}
	return &ecs.StopTaskOutput{Task: ret}, nil
}

func (ctx *EcsServer) ListTasks(_ context.Context, input *ecs.ListTasksInput, _ ...func(options *ecs.Options)) (*ecs.ListTasksOutput, error) {
	var ret []string
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range ctx.Tasks {
		group := fmt.Sprintf("service:%s", *input.ServiceName)
		if *v.Group == group {
			ret = append(ret, *v.TaskArn)
		}
	}
	return &ecs.ListTasksOutput{
		TaskArns: ret,
	}, nil
}

func (ctx *EcsServer) DescribeServices(_ context.Context, input *ecs.DescribeServicesInput, _ ...func(options *ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	var ret []types.Service
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range input.Services {
		if s, ok := ctx.Services[v]; ok {
			ret = append(ret, *s)
		}
	}
	return &ecs.DescribeServicesOutput{
		Services: ret,
	}, nil
}

func (ctx *EcsServer) DescribeTasks(_ context.Context, input *ecs.DescribeTasksInput, _ ...func(options *ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	var ret []types.Task
	for _, task := range ctx.Tasks {
		for _, v := range input.Tasks {
			if *task.TaskArn == v {
				ret = append(ret, *task)
			}
		}
	}
	return &ecs.DescribeTasksOutput{
		Tasks: ret,
	}, nil
}
func (ctx *EcsServer) DescribeContainerInstances(_ context.Context, input *ecs.DescribeContainerInstancesInput, _ ...func(options *ecs.Options)) (*ecs.DescribeContainerInstancesOutput, error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	var ret []types.ContainerInstance
	ec2Id := "i-1234567890abcdefg"
	instance := types.ContainerInstance{
		Ec2InstanceId: &ec2Id,
	}
	ret = append(ret, instance)
	return &ecs.DescribeContainerInstancesOutput{
		ContainerInstances: ret,
	}, nil
}

func (e *EcsServer) DescribeTaskDefinition(ctx context.Context, params *ecs.DescribeTaskDefinitionInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error) {
	td := e.TaskDefinitions.Get(*params.TaskDefinition)
	if td == nil {
		return nil, fmt.Errorf("task definition not found: %s", *params.TaskDefinition)
	}
	return &ecs.DescribeTaskDefinitionOutput{TaskDefinition: td}, nil
}

func (e *EcsServer) ListAttributes(ctx context.Context, params *ecs.ListAttributesInput, optFns ...func(*ecs.Options)) (*ecs.ListAttributesOutput, error) {
	return &ecs.ListAttributesOutput{}, nil
}

func (e *EcsServer) PutAttributes(ctx context.Context, params *ecs.PutAttributesInput, optFns ...func(*ecs.Options)) (*ecs.PutAttributesOutput, error) {
	return nil, nil
}
