package test

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/google/uuid"
)

type MockContext struct {
	Services        map[string]*types.Service
	Tasks           map[string]*types.Task
	TaskDefinitions *TaskDefinitionRepository
	TargetGroups    map[string]struct{}
	mux             sync.Mutex
}

func NewMockContext() *MockContext {
	return &MockContext{
		Services: make(map[string]*types.Service),
		Tasks:    make(map[string]*types.Task),
		TaskDefinitions: &TaskDefinitionRepository{
			families: make(map[string]*TaskDefinitionFamily),
		},
		TargetGroups: make(map[string]struct{}),
	}
}

func (ctx *MockContext) GetTask(id string) (*types.Task, bool) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	o, ok := ctx.Tasks[id]
	return o, ok
}

func (ctx *MockContext) RunningTaskSize() int {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()

	count := 0
	for _, v := range ctx.Tasks {
		if v.LastStatus != nil && *v.LastStatus == "RUNNING" {
			count++
		}
	}

	return count
}

func (ctx *MockContext) GetService(id string) (*types.Service, bool) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	o, ok := ctx.Services[id]
	return o, ok
}

func (ctx *MockContext) ActiveServiceSize() (count int) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range ctx.Services {
		if v.Status != nil && *v.Status == "ACTIVE" {
			count++
		}
	}
	return
}

func (ctx *MockContext) CreateService(c context.Context, input *ecs.CreateServiceInput, _ ...func(options *ecs.Options)) (*ecs.CreateServiceOutput, error) {
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
		ctx.StartTask(c, &ecs.StartTaskInput{
			Cluster:              input.Cluster,
			Group:                aws.String(fmt.Sprintf("service:%s", *input.ServiceName)),
			NetworkConfiguration: input.NetworkConfiguration,
			TaskDefinition:       input.TaskDefinition,
		})
	}
	ctx.mux.Lock()
	ctx.Services[*input.ServiceName].RunningCount = *input.DesiredCount
	ctx.mux.Unlock()
	log.Debugf("%s: running=%d", *input.ServiceName, ret.RunningCount)
	return &ecs.CreateServiceOutput{
		Service: ret,
	}, nil
}

func (ctx *MockContext) UpdateService(c context.Context, input *ecs.UpdateServiceInput, _ ...func(options *ecs.Options)) (*ecs.UpdateServiceOutput, error) {
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

func (ctx *MockContext) DeleteService(c context.Context, input *ecs.DeleteServiceInput, _ ...func(options *ecs.Options)) (*ecs.DeleteServiceOutput, error) {
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

func (ctx *MockContext) RegisterTaskDefinition(_ context.Context, input *ecs.RegisterTaskDefinitionInput, _ ...func(options *ecs.Options)) (*ecs.RegisterTaskDefinitionOutput, error) {
	td, err := ctx.TaskDefinitions.Register(input)
	if err != nil {
		return nil, err
	}
	return &ecs.RegisterTaskDefinitionOutput{TaskDefinition: td}, nil
}

func (ctx *MockContext) StartTask(_ context.Context, input *ecs.StartTaskInput, _ ...func(options *ecs.Options)) (*ecs.StartTaskOutput, error) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	td := ctx.TaskDefinitions.Get(*input.TaskDefinition)
	if td == nil {
		return nil, fmt.Errorf("task definition not found: %s", *input.TaskDefinition)
	}
	taskArn := fmt.Sprintf("arn:aws:ecs:us-west-2:012345678910:task/%s", uuid.New().String())
	attachment := types.Attachment{
		Details: []types.KeyValuePair{
			{
				Name:  aws.String("privateIPv4Address"),
				Value: aws.String("127.0.0.1"),
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

func (ctx *MockContext) RunTask(c context.Context, input *ecs.RunTaskInput, _ ...func(options *ecs.Options)) (*ecs.RunTaskOutput, error) {
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

func (ctx *MockContext) StopTask(_ context.Context, input *ecs.StopTaskInput, _ ...func(options *ecs.Options)) (*ecs.StopTaskOutput, error) {
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

func (ctx *MockContext) ListTasks(_ context.Context, input *ecs.ListTasksInput, _ ...func(options *ecs.Options)) (*ecs.ListTasksOutput, error) {
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

func (ctx *MockContext) DescribeServices(_ context.Context, input *ecs.DescribeServicesInput, _ ...func(options *ecs.Options)) (*ecs.DescribeServicesOutput, error) {
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

func (ctx *MockContext) DescribeTasks(_ context.Context, input *ecs.DescribeTasksInput, _ ...func(options *ecs.Options)) (*ecs.DescribeTasksOutput, error) {
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
func (ctx *MockContext) DescribeContainerInstances(_ context.Context, input *ecs.DescribeContainerInstancesInput, _ ...func(options *ecs.Options)) (*ecs.DescribeContainerInstancesOutput, error) {
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

//

func (ctx *MockContext) DescribeTargetGroups(_ context.Context, input *elbv2.DescribeTargetGroupsInput, _ ...func(options *elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
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
func (ctx *MockContext) DescribeTargetGroupAttibutes(_ context.Context, input *elbv2.DescribeTargetGroupAttributesInput, _ ...func(options *elbv2.Options)) (*elbv2.DescribeTargetGroupAttributesOutput, error) {
	return &elbv2.DescribeTargetGroupAttributesOutput{
		Attributes: []elbv2types.TargetGroupAttribute{
			{
				Key:   aws.String("deregistration_delay.timeout_seconds"),
				Value: aws.String("0"),
			},
		},
	}, nil
}
func (ctx *MockContext) DescribeTargetHealth(_ context.Context, input *elbv2.DescribeTargetHealthInput, _ ...func(options *elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
	if _, ok := ctx.TargetGroups[*input.TargetGroupArn]; !ok {
		return &elbv2.DescribeTargetHealthOutput{
			TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
				{
					Target: &elbv2types.TargetDescription{
						Id:               input.Targets[0].Id,
						Port:             input.Targets[0].Port,
						AvailabilityZone: aws.String("us-west-2"),
					},
					TargetHealth: &elbv2types.TargetHealth{
						State: elbv2types.TargetHealthStateEnumUnused,
					},
				},
			},
		}, nil
	}

	var ret []elbv2types.TargetHealthDescription
	for _, task := range ctx.Tasks {
		if task.LastStatus != nil && *task.LastStatus == "RUNNING" {
			ret = append(ret, elbv2types.TargetHealthDescription{
				Target: &elbv2types.TargetDescription{
					Id:               input.Targets[0].Id,
					Port:             input.Targets[0].Port,
					AvailabilityZone: aws.String("us-west-2"),
				},
				TargetHealth: &elbv2types.TargetHealth{
					State: elbv2types.TargetHealthStateEnumHealthy,
				},
			})
		}
	}
	return &elbv2.DescribeTargetHealthOutput{
		TargetHealthDescriptions: ret,
	}, nil
}

func (ctx *MockContext) RegisterTarget(_ context.Context, input *elbv2.RegisterTargetsInput, _ ...func(options *elbv2.Options)) (*elbv2.RegisterTargetsOutput, error) {
	ctx.TargetGroups[*input.TargetGroupArn] = struct{}{}
	return &elbv2.RegisterTargetsOutput{}, nil
}

func (ctx *MockContext) DeregisterTarget(_ context.Context, input *elbv2.DeregisterTargetsInput, _ ...func(options *elbv2.Options)) (*elbv2.DeregisterTargetsOutput, error) {
	delete(ctx.TargetGroups, *input.TargetGroupArn)
	return &elbv2.DeregisterTargetsOutput{}, nil
}

func (ctx *MockContext) DescribeInstances(_ context.Context, input *ec2.DescribeInstancesInput, _ ...func(options *ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
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

func (ctx *MockContext) DescribeSubnets(_ context.Context, input *ec2.DescribeSubnetsInput, _ ...func(options *ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return &ec2.DescribeSubnetsOutput{
		Subnets: []ec2types.Subnet{{
			AvailabilityZone:              aws.String("us-west-2"),
			AvailabilityZoneId:            nil,
			AvailableIpAddressCount:       nil,
			CidrBlock:                     nil,
			CustomerOwnedIpv4Pool:         nil,
			DefaultForAz:                  nil,
			EnableDns64:                   nil,
			EnableLniAtDeviceIndex:        nil,
			Ipv6CidrBlockAssociationSet:   nil,
			Ipv6Native:                    nil,
			MapCustomerOwnedIpOnLaunch:    nil,
			MapPublicIpOnLaunch:           nil,
			OutpostArn:                    nil,
			OwnerId:                       nil,
			PrivateDnsNameOptionsOnLaunch: nil,
			State:                         ec2types.SubnetStateAvailable,
			SubnetArn:                     nil,
			SubnetId:                      aws.String("subnet-1234567890abcdefg"),
			Tags:                          nil,
			VpcId:                         nil,
		}},
	}, nil
}
