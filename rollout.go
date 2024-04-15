package cage

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

type RollOutResult struct {
	StartTime     time.Time
	EndTime       time.Time
	ServiceIntact bool
}

var WaitDuration = 15 * time.Minute

func (c *cage) RollOut(ctx context.Context) (*RollOutResult, error) {
	ret := &RollOutResult{
		StartTime:     now(),
		ServiceIntact: true,
	}
	var aggregatedError error
	throw := func(err error) (*RollOutResult, error) {
		ret.EndTime = now()
		aggregatedError = err
		return ret, err
	}
	defer func(result *RollOutResult) {
		ret.EndTime = now()
	}(ret)
	var service ecstypes.Service
	if out, err := c.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster: &c.env.Cluster,
		Services: []string{
			c.env.Service,
		},
	}); err != nil {
		log.Errorf("failed to describe current service due to: %s", err.Error())
		return throw(err)
	} else if len(out.Services) == 0 {
		return throw(fmt.Errorf("service '%s' doesn't exist. Run 'cage up' or create service before rolling out", c.env.Service))
	} else {
		service = out.Services[0]
	}
	if *service.Status != "ACTIVE" {
		return throw(fmt.Errorf("😵 '%s' status is '%s'. Stop rolling out", c.env.Service, *service.Status))
	}
	if service.LaunchType == ecstypes.LaunchTypeEc2 && c.env.CanaryInstanceArn == "" {
		return throw(fmt.Errorf("🥺 --canaryInstanceArn is required when LaunchType = 'EC2'"))
	}
	var (
		targetGroupArn *string
	)
	if len(service.LoadBalancers) > 0 {
		targetGroupArn = service.LoadBalancers[0].TargetGroupArn
	}
	log.Infof("ensuring next task definition...")
	nextTaskDefinition, err := c.CreateNextTaskDefinition(ctx)
	if err != nil {
		log.Errorf("failed to register next task definition due to: %s", err)
		return throw(err)
	}
	log.Infof("starting canary task...")
	var canaryTask *StartCanaryTaskOutput
	if o, err := c.StartCanaryTask(ctx, nextTaskDefinition); err != nil {
		log.Errorf("failed to start canary task due to: %s", err)
		return throw(err)
	} else {
		canaryTask = o
	}
	// ensure canary task stopped after rolling out
	defer func(task *StartCanaryTaskOutput, result *RollOutResult) {
		if task == nil {
			return
		}
		if err := c.StopCanaryTask(ctx, canaryTask); err != nil {
			log.Fatalf("failed to stop canary task '%s': %s", *canaryTask.task.TaskArn, err)
		}
		if aggregatedError == nil {
			log.Infof(
				"🐥 service '%s' successfully rolled out to '%s:%d'!",
				c.env.Service, *nextTaskDefinition.Family, nextTaskDefinition.Revision,
			)
		} else {
			log.Errorf(
				"😥 %s", aggregatedError,
			)
		}
	}(canaryTask, ret)

	log.Infof("😷 ensuring canary task container(s) to become healthy...")
	if err := c.waitUntilContainersBecomeHealthy(ctx, *canaryTask.task.TaskArn, nextTaskDefinition); err != nil {
		return throw(err)
	}
	log.Info("🤩 canary task container(s) is healthy!")

	log.Infof("canary task '%s' ensured.", *canaryTask.task.TaskArn)
	if targetGroupArn != nil {
		log.Infof("😷 ensuring canary task to become healthy...")
		if err := c.EnsureTaskHealthy(
			ctx,
			canaryTask.task.TaskArn,
			targetGroupArn,
			canaryTask.targetId,
			canaryTask.targetPort,
		); err != nil {
			return throw(err)
		}
		log.Info("🤩 canary task is healthy!")
	}
	ret.ServiceIntact = false
	log.Infof(
		"updating the task definition of '%s' into '%s:%d'...",
		c.env.Service, *nextTaskDefinition.Family, nextTaskDefinition.Revision,
	)
	if _, err := c.ecs.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:        &c.env.Cluster,
		Service:        &c.env.Service,
		TaskDefinition: nextTaskDefinition.TaskDefinitionArn,
	}); err != nil {
		return throw(err)
	}
	log.Infof("waiting for service '%s' to be stable...", c.env.Service)
	//TODO: avoid stdout sticking while CI

	if err := ecs.NewServicesStableWaiter(c.ecs).Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.env.Cluster,
		Services: []string{c.env.Service},
	}, WaitDuration); err != nil {
		return throw(err)
	}
	log.Infof("🥴 service '%s' has become to be stable!", c.env.Service)
	ret.EndTime = now()
	return ret, nil
}

func (c *cage) EnsureTaskHealthy(
	ctx context.Context,
	taskArn *string,
	tgArn *string,
	targetId *string,
	targetPort *int32,
) error {
	log.Infof("checking the health state of canary task...")
	var unusedCount = 0
	var initialized = false
	var recentState *elbv2types.TargetHealthStateEnum
	for {
		<-newTimer(time.Duration(15) * time.Second).C
		if o, err := c.alb.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: tgArn,
			Targets: []elbv2types.TargetDescription{{
				Id:   targetId,
				Port: targetPort,
			}},
		}); err != nil {
			return err
		} else {
			recentState = GetTargetIsHealthy(o, targetId, targetPort)
			if recentState == nil {
				return fmt.Errorf("'%s' is not registered to the target group '%s'", *targetId, *tgArn)
			}
			log.Infof("canary task '%s' (%s:%d) state is: %s", *taskArn, *targetId, *targetPort, *recentState)
			switch *recentState {
			case "healthy":
				return nil
			case "initial":
				initialized = true
				log.Infof("still checking the state...")
				continue
			case "unused":
				unusedCount++
				if !initialized && unusedCount < 5 {
					continue
				}
			default:
			}
		}
		// unhealthy, draining, unused
		log.Errorf("😨 canary task '%s' is unhealthy", *taskArn)
		return fmt.Errorf(
			"canary task '%s' (%s:%d) hasn't become to be healthy. The most recent state: %s",
			*taskArn, *targetId, *targetPort, *recentState,
		)
	}
}

func GetTargetIsHealthy(o *elbv2.DescribeTargetHealthOutput, targetId *string, targetPort *int32) *elbv2types.TargetHealthStateEnum {
	for _, desc := range o.TargetHealthDescriptions {
		log.Debugf("%+v", desc)
		if *desc.Target.Id == *targetId && *desc.Target.Port == *targetPort {
			return &desc.TargetHealth.State
		}
	}
	return nil
}

func (c *cage) CreateNextTaskDefinition(ctx context.Context) (*ecstypes.TaskDefinition, error) {
	if c.env.TaskDefinitionArn != "" {
		log.Infof("--taskDefinitionArn was set to '%s'. skip registering new task definition.", c.env.TaskDefinitionArn)
		o, err := c.ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &c.env.TaskDefinitionArn,
		})
		if err != nil {
			log.Errorf(
				"failed to describe next task definition '%s' due to: %s",
				c.env.TaskDefinitionArn, err,
			)
			return nil, err
		}
		return o.TaskDefinition, nil
	} else {
		if out, err := c.ecs.RegisterTaskDefinition(ctx, c.env.TaskDefinitionInput); err != nil {
			return nil, err
		} else {
			return out.TaskDefinition, nil
		}
	}
}

func (c *cage) DescribeSubnet(ctx context.Context, subnetId *string) (ec2types.Subnet, error) {
	if o, err := c.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{*subnetId},
	}); err != nil {
		return ec2types.Subnet{}, err
	} else {
		return o.Subnets[0], nil
	}
}

type StartCanaryTaskOutput struct {
	task                ecstypes.Task
	registrationSkipped bool
	targetGroupArn      *string
	availabilityZone    *string
	targetId            *string
	targetPort          *int32
}

func (c *cage) StartCanaryTask(ctx context.Context, nextTaskDefinition *ecstypes.TaskDefinition) (*StartCanaryTaskOutput, error) {
	var service ecstypes.Service
	if o, err := c.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.env.Cluster,
		Services: []string{c.env.Service},
	}); err != nil {
		return nil, err
	} else {
		service = o.Services[0]
	}
	var taskArn *string
	if c.env.CanaryInstanceArn != "" {
		// ec2
		startTask := &ecs.StartTaskInput{
			Cluster:              &c.env.Cluster,
			Group:                aws.String(fmt.Sprintf("cage:canary-task:%s", c.env.Service)),
			NetworkConfiguration: service.NetworkConfiguration,
			TaskDefinition:       nextTaskDefinition.TaskDefinitionArn,
			ContainerInstances:   []string{c.env.CanaryInstanceArn},
		}
		if o, err := c.ecs.StartTask(ctx, startTask); err != nil {
			return nil, err
		} else {
			taskArn = o.Tasks[0].TaskArn
		}
	} else {
		// fargate
		if o, err := c.ecs.RunTask(ctx, &ecs.RunTaskInput{
			Cluster:              &c.env.Cluster,
			Group:                aws.String(fmt.Sprintf("cage:canary-task:%s", c.env.Service)),
			NetworkConfiguration: service.NetworkConfiguration,
			TaskDefinition:       nextTaskDefinition.TaskDefinitionArn,
			LaunchType:           ecstypes.LaunchTypeFargate,
			PlatformVersion:      service.PlatformVersion,
		}); err != nil {
			return nil, err
		} else {
			taskArn = o.Tasks[0].TaskArn
		}
	}
	log.Infof("🥚 waiting for canary task '%s' is running...", *taskArn)
	if err := ecs.NewTasksRunningWaiter(c.ecs).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.env.Cluster,
		Tasks:   []string{*taskArn},
	}, WaitDuration); err != nil {
		return nil, err
	}
	log.Infof("🐣 canary task '%s' is running!️", *taskArn)
	var task ecstypes.Task
	if o, err := c.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.env.Cluster,
		Tasks:   []string{*taskArn},
	}); err != nil {
		return nil, err
	} else {
		task = o.Tasks[0]
	}
	if len(service.LoadBalancers) == 0 {
		log.Infof("no load balancer is attached to service '%s'. skip registration to target group", *service.ServiceName)
		log.Infof("wait %d seconds for ensuring the task goes stable", c.env.CanaryTaskIdleDuration)
		wait := make(chan bool)
		go func() {
			duration := c.env.CanaryTaskIdleDuration
			for duration > 0 {
				log.Infof("still waiting...; %d seconds left", duration)
				wt := 10
				if duration < 10 {
					wt = duration
				}
				<-time.NewTimer(time.Duration(wt) * time.Second).C
				duration -= 10
			}
			wait <- true
		}()
		<-wait
		o, err := c.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &c.env.Cluster,
			Tasks:   []string{*taskArn},
		})
		if err != nil {
			return nil, err
		}
		task := o.Tasks[0]
		if *task.LastStatus != "RUNNING" {
			return nil, fmt.Errorf("😫 canary task has stopped: %s", *task.StoppedReason)
		}
		return &StartCanaryTaskOutput{
			task:                task,
			registrationSkipped: true,
		}, nil
	}
	var targetId *string
	var targetPort *int32
	var subnet ec2types.Subnet
	for _, container := range nextTaskDefinition.ContainerDefinitions {
		if *container.Name == *service.LoadBalancers[0].ContainerName {
			targetPort = container.PortMappings[0].HostPort
		}
	}
	if task.LaunchType == ecstypes.LaunchTypeFargate {
		details := task.Attachments[0].Details
		var subnetId *string
		var privateIp *string
		for _, v := range details {
			if *v.Name == "subnetId" {
				subnetId = v.Value
			} else if *v.Name == "privateIPv4Address" {
				privateIp = v.Value
			}
		}
		if o, err := c.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: []string{*subnetId},
		}); err != nil {
			return nil, err
		} else {
			subnet = o.Subnets[0]
		}
		targetId = privateIp
		log.Infof("canary task was placed: privateIp = '%s', hostPort = '%d', az = '%s'", *targetId, *targetPort, *subnet.AvailabilityZone)
	} else {
		var containerInstance ecstypes.ContainerInstance
		if outputs, err := c.ecs.DescribeContainerInstances(ctx, &ecs.DescribeContainerInstancesInput{
			Cluster:            &c.env.Cluster,
			ContainerInstances: []string{c.env.CanaryInstanceArn},
		}); err != nil {
			return nil, err
		} else {
			containerInstance = outputs.ContainerInstances[0]
		}
		if o, err := c.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{*containerInstance.Ec2InstanceId},
		}); err != nil {
			return nil, err
		} else if sn, err := c.DescribeSubnet(ctx, o.Reservations[0].Instances[0].SubnetId); err != nil {
			return nil, err
		} else {
			targetId = containerInstance.Ec2InstanceId
			subnet = sn
		}
		log.Infof("canary task was placed: instanceId = '%s', hostPort = '%d', az = '%s'", *targetId, *targetPort, *subnet.AvailabilityZone)
	}
	if _, err := c.alb.RegisterTargets(ctx, &elbv2.RegisterTargetsInput{
		TargetGroupArn: service.LoadBalancers[0].TargetGroupArn,
		Targets: []elbv2types.TargetDescription{{
			AvailabilityZone: subnet.AvailabilityZone,
			Id:               targetId,
			Port:             targetPort,
		}},
	}); err != nil {
		return nil, err
	}
	return &StartCanaryTaskOutput{
		targetGroupArn: service.LoadBalancers[0].TargetGroupArn,
		targetId:       targetId,
		targetPort:     targetPort,
		task:           task,
	}, nil
}

func (c *cage) waitUntilContainersBecomeHealthy(ctx context.Context, taskArn string, nextTaskDefinition *ecstypes.TaskDefinition) error {
	containerHasHealthChecks := map[string]struct{}{}
	for _, definition := range nextTaskDefinition.ContainerDefinitions {
		if definition.HealthCheck != nil {
			containerHasHealthChecks[*definition.Name] = struct{}{}
		}
	}

	for count := 0; count < 10; count++ {
		<-newTimer(time.Duration(15) * time.Second).C
		log.Infof("canary task '%s' waits until %d container(s) become healthy", taskArn, len(containerHasHealthChecks))
		if o, err := c.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &c.env.Cluster,
			Tasks:   []string{taskArn},
		}); err != nil {
			return err
		} else {
			task := o.Tasks[0]
			if *task.LastStatus != "RUNNING" {
				return fmt.Errorf("😫 canary task has stopped: %s", *task.StoppedReason)
			}

			for _, container := range task.Containers {
				if _, ok := containerHasHealthChecks[*container.Name]; !ok {
					continue
				}
				if container.HealthStatus != ecstypes.HealthStatusHealthy {
					log.Infof("container '%s' is not healthy: %s", *container.Name, container.HealthStatus)
					continue
				}
				delete(containerHasHealthChecks, *container.Name)
			}
			if len(containerHasHealthChecks) == 0 {
				return nil
			}
		}
	}
	return fmt.Errorf("😨 canary task hasn't become to be healthy")
}

func (c *cage) StopCanaryTask(ctx context.Context, input *StartCanaryTaskOutput) error {
	if input.registrationSkipped {
		log.Info("no load balancer is attached to service. Skip deregisteration.")
	} else {
		log.Infof("deregistering the canary task from target group '%s'...", *input.targetId)
		if _, err := c.alb.DeregisterTargets(ctx, &elbv2.DeregisterTargetsInput{
			TargetGroupArn: input.targetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				AvailabilityZone: input.availabilityZone,
				Id:               input.targetId,
				Port:             input.targetPort,
			}},
		}); err != nil {
			return err
		}
		if err := elbv2.NewTargetDeregisteredWaiter(c.alb).Wait(ctx, &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: input.targetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				AvailabilityZone: input.availabilityZone,
				Id:               input.targetId,
				Port:             input.targetPort,
			}},
		}, WaitDuration); err != nil {
			return err
		}
		log.Infof(
			"canary task '%s' has successfully been deregistered from target group '%s'",
			*input.task.TaskArn, *input.targetId,
		)
	}

	log.Infof("stopping the canary task '%s'...", *input.task.TaskArn)
	if _, err := c.ecs.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &c.env.Cluster,
		Task:    input.task.TaskArn,
	}); err != nil {
		return err
	}
	if err := ecs.NewTasksStoppedWaiter(c.ecs).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.env.Cluster,
		Tasks:   []string{*input.task.TaskArn},
	}, WaitDuration); err != nil {
		return err
	}
	log.Infof("canary task '%s' has successfully been stopped", *input.task.TaskArn)
	return nil
}
