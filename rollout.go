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
	"golang.org/x/xerrors"
)

type RollOutInput struct {
	// UpdateService is a flag to update service with changed configurations except for task definition
	UpdateService bool
}

type RollOutResult struct {
	StartTime     time.Time
	EndTime       time.Time
	ServiceIntact bool
}

func (c *cage) RollOut(ctx context.Context, input *RollOutInput) (*RollOutResult, error) {
	ret := &RollOutResult{
		StartTime:     c.Time.Now(),
		ServiceIntact: true,
	}
	var aggregatedError error
	throw := func(err error) (*RollOutResult, error) {
		ret.EndTime = c.Time.Now()
		aggregatedError = err
		return ret, err
	}
	defer func(result *RollOutResult) {
		ret.EndTime = c.Time.Now()
	}(ret)
	var service ecstypes.Service
	if out, err := c.Ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster: &c.Env.Cluster,
		Services: []string{
			c.Env.Service,
		},
	}); err != nil {
		log.Errorf("failed to describe current service due to: %s", err)
		return throw(err)
	} else if len(out.Services) == 0 {
		return throw(xerrors.Errorf("service '%s' doesn't exist. Run 'cage up' or create service before rolling out", c.Env.Service))
	} else {
		service = out.Services[0]
	}
	if *service.Status != "ACTIVE" {
		return throw(xerrors.Errorf("😵 '%s' status is '%s'. Stop rolling out", c.Env.Service, *service.Status))
	}
	if service.LaunchType == ecstypes.LaunchTypeEc2 && c.Env.CanaryInstanceArn == "" {
		return throw(xerrors.Errorf("🥺 --canaryInstanceArn is required when LaunchType = 'EC2'"))
	}
	log.Infof("ensuring next task definition...")
	var nextTaskDefinition *ecstypes.TaskDefinition
	if o, err := c.CreateNextTaskDefinition(ctx); err != nil {
		log.Errorf("failed to register next task definition due to: %s", err)
		return throw(err)
	} else {
		nextTaskDefinition = o
	}
	if input.UpdateService {
		log.Info("--updateService flag is set. use provided service configurations for canary test instead of current service")
	}
	log.Infof("starting canary task...")
	canaryTask, startCanaryTaskErr := c.StartCanaryTask(ctx, nextTaskDefinition, input)
	// ensure canary task stopped after rolling out either success or failure
	defer func(canaryTask *CanaryTask, result *RollOutResult) {
		if canaryTask.taskArn == nil {
			return
		}
		if err := c.StopCanaryTask(ctx, canaryTask); err != nil {
			log.Fatalf("failed to stop canary task '%s': %s", *canaryTask.taskArn, err)
		}
		if aggregatedError == nil {
			log.Infof(
				"🐥 service '%s' successfully rolled out to '%s:%d'!",
				c.Env.Service, *nextTaskDefinition.Family, nextTaskDefinition.Revision,
			)
		} else {
			log.Errorf("😥 %s", aggregatedError)
		}
	}(&canaryTask, ret)
	if startCanaryTaskErr != nil {
		log.Errorf("failed to start canary task due to: %s", startCanaryTaskErr)
		return throw(startCanaryTaskErr)
	}
	log.Infof("😷 ensuring canary task container(s) to become healthy...")
	if err := c.waitUntilContainersBecomeHealthy(ctx, *canaryTask.taskArn, nextTaskDefinition); err != nil {
		return throw(err)
	}
	log.Info("🤩 canary task container(s) is healthy!")
	log.Infof("canary task '%s' ensured.", *canaryTask.taskArn)
	if canaryTask.target != nil {
		log.Infof("😷 ensuring canary task to become healthy...")
		if err := c.EnsureTaskHealthy(
			ctx,
			*canaryTask.taskArn,
			canaryTask.target,
		); err != nil {
			return throw(err)
		}
		log.Info("🤩 canary task is healthy!")
	}
	ret.ServiceIntact = false
	log.Infof(
		"updating the task definition of '%s' into '%s:%d'...",
		c.Env.Service, *nextTaskDefinition.Family, nextTaskDefinition.Revision,
	)
	updateInput := &ecs.UpdateServiceInput{
		Cluster:        &c.Env.Cluster,
		Service:        &c.Env.Service,
		TaskDefinition: nextTaskDefinition.TaskDefinitionArn,
	}
	if input.UpdateService {
		updateInput.LoadBalancers = c.Env.ServiceDefinitionInput.LoadBalancers
		updateInput.NetworkConfiguration = c.Env.ServiceDefinitionInput.NetworkConfiguration
		updateInput.ServiceConnectConfiguration = c.Env.ServiceDefinitionInput.ServiceConnectConfiguration
		updateInput.ServiceRegistries = c.Env.ServiceDefinitionInput.ServiceRegistries
		updateInput.PlatformVersion = c.Env.ServiceDefinitionInput.PlatformVersion
		updateInput.VolumeConfigurations = c.Env.ServiceDefinitionInput.VolumeConfigurations
	}
	if _, err := c.Ecs.UpdateService(ctx, updateInput); err != nil {
		return throw(err)
	}
	log.Infof("waiting for service '%s' to be stable...", c.Env.Service)
	if err := ecs.NewServicesStableWaiter(c.Ecs).Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.Env.Cluster,
		Services: []string{c.Env.Service},
	}, c.MaxWait); err != nil {
		return throw(err)
	}
	log.Infof("🥴 service '%s' has become to be stable!", c.Env.Service)
	ret.EndTime = c.Time.Now()
	return ret, nil
}

func (c *cage) EnsureTaskHealthy(
	ctx context.Context,
	taskArn string,
	p *CanaryTarget,
) error {
	log.Infof("checking the health state of canary task...")
	var unusedCount = 0
	var initialized = false
	var recentState *elbv2types.TargetHealthStateEnum
	for {
		<-c.Time.NewTimer(time.Duration(15) * time.Second).C
		if o, err := c.Alb.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: &p.targetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				Id:               &p.targetId,
				Port:             &p.targetPort,
				AvailabilityZone: &p.availabilityZone,
			}},
		}); err != nil {
			return err
		} else {
			recentState = GetTargetIsHealthy(o, &p.targetId, &p.targetPort)
			if recentState == nil {
				return xerrors.Errorf("'%s' is not registered to the target group '%s'", p.targetId, p.targetGroupArn)
			}
			log.Infof("canary task '%s' (%s:%d) state is: %s", taskArn, p.targetId, p.targetPort, *recentState)
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
		log.Errorf("😨 canary task '%s' is unhealthy", taskArn)
		return xerrors.Errorf(
			"canary task '%s' (%s:%d) hasn't become to be healthy. The most recent state: %s",
			taskArn, p.targetId, p.targetPort, *recentState,
		)
	}
}

func GetTargetIsHealthy(o *elbv2.DescribeTargetHealthOutput, targetId *string, targetPort *int32) *elbv2types.TargetHealthStateEnum {
	for _, desc := range o.TargetHealthDescriptions {
		if *desc.Target.Id == *targetId && *desc.Target.Port == *targetPort {
			return &desc.TargetHealth.State
		}
	}
	return nil
}

func (c *cage) DescribeSubnet(ctx context.Context, subnetId *string) (ec2types.Subnet, error) {
	if o, err := c.Ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{*subnetId},
	}); err != nil {
		return ec2types.Subnet{}, err
	} else {
		return o.Subnets[0], nil
	}
}

type CanaryTarget struct {
	targetGroupArn   string
	targetId         string
	targetPort       int32
	availabilityZone string
}

type CanaryTask struct {
	taskArn *string
	target  *CanaryTarget
}

func (c *cage) StartCanaryTask(
	ctx context.Context,
	nextTaskDefinition *ecstypes.TaskDefinition,
	input *RollOutInput,
) (CanaryTask, error) {
	var networkConfiguration *ecstypes.NetworkConfiguration
	var platformVersion *string
	var loadBalancers []ecstypes.LoadBalancer
	var result CanaryTask
	if input.UpdateService {
		networkConfiguration = c.Env.ServiceDefinitionInput.NetworkConfiguration
		platformVersion = c.Env.ServiceDefinitionInput.PlatformVersion
		loadBalancers = c.Env.ServiceDefinitionInput.LoadBalancers
	} else {
		if o, err := c.Ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  &c.Env.Cluster,
			Services: []string{c.Env.Service},
		}); err != nil {
			return result, err
		} else {
			service := o.Services[0]
			networkConfiguration = service.NetworkConfiguration
			platformVersion = service.PlatformVersion
			loadBalancers = service.LoadBalancers
		}
	}
	// Phase 1: Start canary task
	var taskArn *string
	if c.Env.CanaryInstanceArn != "" {
		// ec2
		startTask := &ecs.StartTaskInput{
			Cluster:              &c.Env.Cluster,
			Group:                aws.String(fmt.Sprintf("cage:canary-task:%s", c.Env.Service)),
			NetworkConfiguration: networkConfiguration,
			TaskDefinition:       nextTaskDefinition.TaskDefinitionArn,
			ContainerInstances:   []string{c.Env.CanaryInstanceArn},
		}
		if o, err := c.Ecs.StartTask(ctx, startTask); err != nil {
			return result, err
		} else {
			taskArn = o.Tasks[0].TaskArn
		}
	} else {
		// fargate
		if o, err := c.Ecs.RunTask(ctx, &ecs.RunTaskInput{
			Cluster:              &c.Env.Cluster,
			Group:                aws.String(fmt.Sprintf("cage:canary-task:%s", c.Env.Service)),
			NetworkConfiguration: networkConfiguration,
			TaskDefinition:       nextTaskDefinition.TaskDefinitionArn,
			LaunchType:           ecstypes.LaunchTypeFargate,
			PlatformVersion:      platformVersion,
		}); err != nil {
			return result, err
		} else {
			taskArn = o.Tasks[0].TaskArn
		}
	}
	result.taskArn = taskArn
	// Phase 2: Wait until canary task is running
	log.Infof("🥚 waiting for canary task '%s' is running...", *taskArn)
	if err := ecs.NewTasksRunningWaiter(c.Ecs).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.Env.Cluster,
		Tasks:   []string{*taskArn},
	}, c.MaxWait); err != nil {
		return result, err
	}
	log.Infof("🐣 canary task '%s' is running!", *taskArn)
	if len(loadBalancers) == 0 {
		log.Infof("no load balancer is attached to service '%s'. skip registration to target group", c.Env.Service)
		log.Infof("wait %d seconds for ensuring the task goes stable", c.Env.CanaryTaskIdleDuration)
		wait := make(chan bool)
		go func() {
			duration := c.Env.CanaryTaskIdleDuration
			for duration > 0 {
				log.Infof("still waiting...; %d seconds left", duration)
				wt := 10
				if duration < 10 {
					wt = duration
				}
				<-c.Time.NewTimer(time.Duration(wt) * time.Second).C
				duration -= 10
			}
			wait <- true
		}()
		<-wait
		o, err := c.Ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &c.Env.Cluster,
			Tasks:   []string{*taskArn},
		})
		if err != nil {
			return result, err
		}
		task := o.Tasks[0]
		if *task.LastStatus != "RUNNING" {
			return result, xerrors.Errorf("😫 canary task has stopped: %s", *task.StoppedReason)
		}
		return result, nil
	}
	// Phase 3: Get task details after network interface is attached
	var task ecstypes.Task
	if o, err := c.Ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.Env.Cluster,
		Tasks:   []string{*taskArn},
	}); err != nil {
		return result, err
	} else {
		task = o.Tasks[0]
	}
	var targetId *string
	var targetPort *int32
	var subnet ec2types.Subnet
	for _, container := range nextTaskDefinition.ContainerDefinitions {
		if *container.Name == *loadBalancers[0].ContainerName {
			targetPort = container.PortMappings[0].HostPort
		}
	}
	if targetPort == nil {
		return result, xerrors.Errorf("couldn't find host port in container definition")
	}
	if c.Env.CanaryInstanceArn == "" { // Fargate
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
		if subnetId == nil || privateIp == nil {
			return result, xerrors.Errorf("couldn't find subnetId or privateIPv4Address in task details")
		}
		if o, err := c.Ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: []string{*subnetId},
		}); err != nil {
			return result, err
		} else {
			subnet = o.Subnets[0]
		}
		targetId = privateIp
		log.Infof("canary task was placed: privateIp = '%s', hostPort = '%d', az = '%s'", *targetId, *targetPort, *subnet.AvailabilityZone)
	} else {
		var containerInstance ecstypes.ContainerInstance
		if outputs, err := c.Ecs.DescribeContainerInstances(ctx, &ecs.DescribeContainerInstancesInput{
			Cluster:            &c.Env.Cluster,
			ContainerInstances: []string{c.Env.CanaryInstanceArn},
		}); err != nil {
			return result, err
		} else {
			containerInstance = outputs.ContainerInstances[0]
		}
		if o, err := c.Ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{*containerInstance.Ec2InstanceId},
		}); err != nil {
			return result, err
		} else if sn, err := c.DescribeSubnet(ctx, o.Reservations[0].Instances[0].SubnetId); err != nil {
			return result, err
		} else {
			targetId = containerInstance.Ec2InstanceId
			subnet = sn
		}
		log.Infof("canary task was placed: instanceId = '%s', hostPort = '%d', az = '%s'", *targetId, *targetPort, *subnet.AvailabilityZone)
	}
	if _, err := c.Alb.RegisterTargets(ctx, &elbv2.RegisterTargetsInput{
		TargetGroupArn: loadBalancers[0].TargetGroupArn,
		Targets: []elbv2types.TargetDescription{{
			AvailabilityZone: subnet.AvailabilityZone,
			Id:               targetId,
			Port:             targetPort,
		}},
	}); err != nil {
		return result, err
	}
	result.target = &CanaryTarget{
		targetGroupArn:   *loadBalancers[0].TargetGroupArn,
		targetId:         *targetId,
		targetPort:       *targetPort,
		availabilityZone: *subnet.AvailabilityZone,
	}
	return result, nil
}

func (c *cage) waitUntilContainersBecomeHealthy(ctx context.Context, taskArn string, nextTaskDefinition *ecstypes.TaskDefinition) error {
	containerHasHealthChecks := map[string]struct{}{}
	for _, definition := range nextTaskDefinition.ContainerDefinitions {
		if definition.HealthCheck != nil {
			containerHasHealthChecks[*definition.Name] = struct{}{}
		}
	}

	for count := 0; count < 10; count++ {
		<-c.Time.NewTimer(time.Duration(15) * time.Second).C
		log.Infof("canary task '%s' waits until %d container(s) become healthy", taskArn, len(containerHasHealthChecks))
		if o, err := c.Ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &c.Env.Cluster,
			Tasks:   []string{taskArn},
		}); err != nil {
			return err
		} else {
			task := o.Tasks[0]
			if *task.LastStatus != "RUNNING" {
				return xerrors.Errorf("😫 canary task has stopped: %s", *task.StoppedReason)
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
	return xerrors.Errorf("😨 canary task hasn't become to be healthy")
}

func (c *cage) StopCanaryTask(ctx context.Context, input *CanaryTask) error {
	if input.target == nil {
		log.Info("no load balancer is attached to service. Skip deregisteration.")
	} else {
		log.Infof("deregistering the canary task from target group '%s'...", input.target.targetId)
		if _, err := c.Alb.DeregisterTargets(ctx, &elbv2.DeregisterTargetsInput{
			TargetGroupArn: &input.target.targetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				AvailabilityZone: &input.target.availabilityZone,
				Id:               &input.target.targetId,
				Port:             &input.target.targetPort,
			}},
		}); err != nil {
			return err
		}
		if err := elbv2.NewTargetDeregisteredWaiter(c.Alb).Wait(ctx, &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: &input.target.targetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				AvailabilityZone: &input.target.availabilityZone,
				Id:               &input.target.targetId,
				Port:             &input.target.targetPort,
			}},
		}, c.MaxWait); err != nil {
			return err
		}
		log.Infof(
			"canary task '%s' has successfully been deregistered from target group '%s'",
			*input.taskArn, input.target.targetId,
		)
	}

	log.Infof("stopping the canary task '%s'...", *input.taskArn)
	if _, err := c.Ecs.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &c.Env.Cluster,
		Task:    input.taskArn,
	}); err != nil {
		return err
	}
	if err := ecs.NewTasksStoppedWaiter(c.Ecs).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.Env.Cluster,
		Tasks:   []string{*input.taskArn},
	}, c.MaxWait); err != nil {
		return err
	}
	log.Infof("canary task '%s' has successfully been stopped", *input.taskArn)
	return nil
}
