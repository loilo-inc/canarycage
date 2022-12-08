package cage

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

type RollOutResult struct {
	StartTime     time.Time
	EndTime       time.Time
	ServiceIntact bool
}

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
	var service *ecs.Service
	if out, err := c.ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: &c.env.Cluster,
		Services: []*string{
			&c.env.Service,
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
	if *service.LaunchType == "EC2" && c.env.CanaryInstanceArn == "" {
		return throw(fmt.Errorf("🥺 --canaryInstanceArn is required when LaunchType = 'EC2'"))
	}
	var (
		targetGroupArn *string
	)
	if len(service.LoadBalancers) > 0 {
		targetGroupArn = service.LoadBalancers[0].TargetGroupArn
	}
	log.Infof("ensuring next task definition...")
	nextTaskDefinition, err := c.CreateNextTaskDefinition()
	if err != nil {
		log.Errorf("failed to register next task definition due to: %s", err)
		return throw(err)
	}
	log.Infof("starting canary task...")
	var canaryTask *StartCanaryTaskOutput
	if o, err := c.StartCanaryTask(nextTaskDefinition); err != nil {
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
		if err := c.StopCanaryTask(canaryTask); err != nil {
			log.Fatalf("failed to stop canary task '%s': %s", *canaryTask.task.TaskArn, err)
		}
		if aggregatedError == nil {
			log.Infof(
				"🐥 service '%s' successfully rolled out to '%s:%d'!",
				c.env.Service, *nextTaskDefinition.Family, *nextTaskDefinition.Revision,
			)
		} else {
			log.Errorf(
				"😥 %s", aggregatedError,
			)
		}
	}(canaryTask, ret)
	log.Infof("canary task '%s' ensured.", *canaryTask.task.TaskArn)
	if targetGroupArn != nil {
		log.Infof("😷 ensuring canary task to become healthy...")
		if err := c.EnsureTaskHealthy(
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
		"updating '%s' 's task definition to '%s:%d'...",
		c.env.Service, *nextTaskDefinition.Family, *nextTaskDefinition.Revision,
	)
	if _, err := c.ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:        &c.env.Cluster,
		Service:        &c.env.Service,
		TaskDefinition: nextTaskDefinition.TaskDefinitionArn,
	}); err != nil {
		return throw(err)
	}
	log.Infof("waiting for service '%s' to be stable...", c.env.Service)
	waiterOption := request.WaiterOption(func(waiter *request.Waiter) {
		waiter.MaxAttempts = 60
	})
	//TODO: avoid stdout sticking while CI
	if err := c.ecs.WaitUntilServicesStableWithContext(aws.BackgroundContext(), &ecs.DescribeServicesInput{
		Cluster:  &c.env.Cluster,
		Services: []*string{&c.env.Service},
	}, waiterOption); err != nil {
		return throw(err)
	}
	log.Infof("🥴 service '%s' has become to be stable!", c.env.Service)
	ret.EndTime = now()
	return ret, nil
}

func (c *cage) EnsureTaskHealthy(
	taskArn *string,
	tgArn *string,
	targetId *string,
	targetPort *int64,
) error {
	log.Infof("checking canary task's health state...")
	var unusedCount = 0
	var initialized = false
	var recentState *string
	for {
		<-newTimer(time.Duration(15) * time.Second).C
		if o, err := c.alb.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: tgArn,
			Targets: []*elbv2.TargetDescription{{
				Id:   targetId,
				Port: targetPort,
			}},
		}); err != nil {
			return err
		} else {
			recentState = GetTargetIsHealthy(o, targetId, targetPort)
			if recentState == nil {
				return fmt.Errorf("'%s' is not registered to target group '%s'", *targetId, *tgArn)
			}
			log.Infof("canary task '%s' (%s:%d) state is: %s", *taskArn, *targetId, *targetPort, *recentState)
			switch *recentState {
			case "healthy":
				return nil
			case "initial":
				initialized = true
				log.Infof("still checking state...")
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
			"canary task '%s' (%s:%d) hasn't become to be healthy. recent state: %s",
			*taskArn, *targetId, *targetPort, *recentState,
		)
	}
}

func GetTargetIsHealthy(o *elbv2.DescribeTargetHealthOutput, targetId *string, targetPort *int64) *string {
	for _, desc := range o.TargetHealthDescriptions {
		log.Debugf("%+v", desc)
		if *desc.Target.Id == *targetId && *desc.Target.Port == *targetPort {
			return desc.TargetHealth.State
		}
	}
	return nil
}

func (c *cage) CreateNextTaskDefinition() (*ecs.TaskDefinition, error) {
	if c.env.TaskDefinitionArn != "" {
		log.Infof("--taskDefinitionArn was set to '%s'. skip registering new task definition.", c.env.TaskDefinitionArn)
		o, err := c.ecs.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
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
		if out, err := c.ecs.RegisterTaskDefinition(c.env.TaskDefinitionInput); err != nil {
			return nil, err
		} else {
			return out.TaskDefinition, nil
		}
	}
}

func (c *cage) DescribeSubnet(subnetId *string) (*ec2.Subnet, error) {
	if o, err := c.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		SubnetIds: []*string{subnetId},
	}); err != nil {
		return nil, err
	} else {
		return o.Subnets[0], nil
	}
}

type StartCanaryTaskOutput struct {
	task                *ecs.Task
	registrationSkipped bool
	targetGroupArn      *string
	availabilityZone    *string
	targetId            *string
	targetPort          *int64
}

func (c *cage) StartCanaryTask(nextTaskDefinition *ecs.TaskDefinition) (*StartCanaryTaskOutput, error) {
	var service *ecs.Service
	if o, err := c.ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  &c.env.Cluster,
		Services: []*string{&c.env.Service},
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
			ContainerInstances:   []*string{&c.env.CanaryInstanceArn},
		}
		if o, err := c.ecs.StartTask(startTask); err != nil {
			return nil, err
		} else {
			taskArn = o.Tasks[0].TaskArn
		}
	} else {
		// fargate
		if o, err := c.ecs.RunTask(&ecs.RunTaskInput{
			Cluster:              &c.env.Cluster,
			Group:                aws.String(fmt.Sprintf("cage:canary-task:%s", c.env.Service)),
			NetworkConfiguration: service.NetworkConfiguration,
			TaskDefinition:       nextTaskDefinition.TaskDefinitionArn,
			LaunchType:           aws.String("FARGATE"),
			PlatformVersion:      service.PlatformVersion,
		}); err != nil {
			return nil, err
		} else {
			taskArn = o.Tasks[0].TaskArn
		}
	}
	log.Infof("🥚 waiting for canary task '%s' is running...", *taskArn)
	if err := c.ecs.WaitUntilTasksRunning(&ecs.DescribeTasksInput{
		Cluster: &c.env.Cluster,
		Tasks:   []*string{taskArn},
	}); err != nil {
		return nil, err
	}
	log.Infof("🐣 canary task '%s' is running!️", *taskArn)
	var task *ecs.Task
	if o, err := c.ecs.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: &c.env.Cluster,
		Tasks:   []*string{taskArn},
	}); err != nil {
		return nil, err
	} else {
		task = o.Tasks[0]
	}
	if len(service.LoadBalancers) == 0 {
		log.Infof("No load balancer is attached to service '%s'. skip registration to target group", *service.ServiceName)
		log.Infof("Waiting for %s to check task doesn't failed to start and to be stable", c.env.CanaryTaskIdleDuration)
		wait := make(chan bool)
		go func() {
			duration := c.env.CanaryTaskIdleDuration
			for duration > 0 {
				log.Infof("Still waiting...%ds left", duration)
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
		o, err := c.ecs.DescribeTasks(&ecs.DescribeTasksInput{
			Cluster: &c.env.Cluster,
			Tasks:   []*string{taskArn},
		})
		if err != nil {
			return nil, err
		}
		task := o.Tasks[0]
		if *task.LastStatus != "RUNNING" {
			return nil, fmt.Errorf("😫 Canary task has been stopped: %s", *task.StoppedReason)
		}
		return &StartCanaryTaskOutput{
			task:                task,
			registrationSkipped: true,
		}, nil
	}
	var targetId *string
	var targetPort *int64
	var subnet *ec2.Subnet
	for _, container := range nextTaskDefinition.ContainerDefinitions {
		if *container.Name == *service.LoadBalancers[0].ContainerName {
			targetPort = container.PortMappings[0].HostPort
		}
	}
	if *task.LaunchType == "FARGATE" {
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
		if o, err := c.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
			SubnetIds: []*string{subnetId},
		}); err != nil {
			return nil, err
		} else {
			subnet = o.Subnets[0]
		}
		targetId = privateIp
		log.Infof("canary task was placed: privateIp = '%s', hostPort = '%d', az = '%s'", *targetId, *targetPort, *subnet.AvailabilityZone)
	} else {
		var containerInstance *ecs.ContainerInstance
		if outputs, err := c.ecs.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            &c.env.Cluster,
			ContainerInstances: []*string{&c.env.CanaryInstanceArn},
		}); err != nil {
			return nil, err
		} else {
			containerInstance = outputs.ContainerInstances[0]
		}
		if o, err := c.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{containerInstance.Ec2InstanceId},
		}); err != nil {
			return nil, err
		} else if sn, err := c.DescribeSubnet(o.Reservations[0].Instances[0].SubnetId); err != nil {
			return nil, err
		} else {
			targetId = containerInstance.Ec2InstanceId
			subnet = sn
		}
		log.Infof("canary task was placed: instanceId = '%s', hostPort = '%d', az = '%s'", *targetId, *targetPort, *subnet.AvailabilityZone)
	}
	if _, err := c.alb.RegisterTargets(&elbv2.RegisterTargetsInput{
		TargetGroupArn: service.LoadBalancers[0].TargetGroupArn,
		Targets: []*elbv2.TargetDescription{{
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

func (c *cage) StopCanaryTask(input *StartCanaryTaskOutput) error {
	if input.registrationSkipped {
		log.Info("No load balancer attached to service. Skip de-registering.")
	} else {
		log.Infof("De-registering canary task from target group '%s'...", *input.targetId)
		if _, err := c.alb.DeregisterTargets(&elbv2.DeregisterTargetsInput{
			TargetGroupArn: input.targetGroupArn,
			Targets: []*elbv2.TargetDescription{{
				AvailabilityZone: input.availabilityZone,
				Id:               input.targetId,
				Port:             input.targetPort,
			}},
		}); err != nil {
			return err
		}
		if err := c.alb.WaitUntilTargetDeregistered(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: input.targetGroupArn,
			Targets: []*elbv2.TargetDescription{{
				AvailabilityZone: input.availabilityZone,
				Id:               input.targetId,
				Port:             input.targetPort,
			}},
		}); err != nil {
			return err
		}
		log.Infof(
			"Canary task '%s' has successfully been de-registered from target group '%s'",
			*input.task.TaskArn, *input.targetId,
		)
	}

	log.Infof("Stopping canary task '%s'...", *input.task.TaskArn)
	if _, err := c.ecs.StopTask(&ecs.StopTaskInput{
		Cluster: &c.env.Cluster,
		Task:    input.task.TaskArn,
	}); err != nil {
		return err
	}
	log.Infof("Canary task '%s' has successfully been stopped", *input.task.TaskArn)
	if err := c.ecs.WaitUntilTasksStopped(&ecs.DescribeTasksInput{
		Cluster: &c.env.Cluster,
		Tasks:   []*string{input.task.TaskArn},
	}); err != nil {
		return err
	}
	return nil
}
