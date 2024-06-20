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

type CanaryTarget struct {
	targetGroupArn   *string
	targetId         *string
	targetPort       *int32
	availabilityZone *string
}

type CanaryTask struct {
	*cage
	td                   *ecstypes.TaskDefinition
	lb                   *ecstypes.LoadBalancer
	networkConfiguration *ecstypes.NetworkConfiguration
	platformVersion      *string
	taskArn              *string
	target               *CanaryTarget
}

func (c *CanaryTask) Start(ctx context.Context) error {
	if c.Env.CanaryInstanceArn != "" {
		// ec2
		startTask := &ecs.StartTaskInput{
			Cluster:              &c.Env.Cluster,
			Group:                aws.String(fmt.Sprintf("cage:canary-task:%s", c.Env.Service)),
			NetworkConfiguration: c.networkConfiguration,
			TaskDefinition:       c.td.TaskDefinitionArn,
			ContainerInstances:   []string{c.Env.CanaryInstanceArn},
		}
		if o, err := c.Ecs.StartTask(ctx, startTask); err != nil {
			return err
		} else {
			c.taskArn = o.Tasks[0].TaskArn
		}
	} else {
		// fargate
		if o, err := c.Ecs.RunTask(ctx, &ecs.RunTaskInput{
			Cluster:              &c.Env.Cluster,
			Group:                aws.String(fmt.Sprintf("cage:canary-task:%s", c.Env.Service)),
			NetworkConfiguration: c.networkConfiguration,
			TaskDefinition:       c.td.TaskDefinitionArn,
			LaunchType:           ecstypes.LaunchTypeFargate,
			PlatformVersion:      c.platformVersion,
		}); err != nil {
			return err
		} else {
			c.taskArn = o.Tasks[0].TaskArn
		}
	}
	return nil
}

func (c *CanaryTask) Wait(ctx context.Context) error {
	log.Infof("🥚 waiting for canary task '%s' is running...", *c.taskArn)
	if err := ecs.NewTasksRunningWaiter(c.Ecs).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.Env.Cluster,
		Tasks:   []string{*c.taskArn},
	}, c.MaxWait); err != nil {
		return err
	}
	log.Infof("🐣 canary task '%s' is running!", *c.taskArn)
	if err := c.waitUntilHealthCeheckPassed(ctx); err != nil {
		return err
	}
	log.Info("🤩 canary task container(s) is healthy!")
	log.Infof("canary task '%s' ensured.", *c.taskArn)
	if c.lb == nil {
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
			Tasks:   []string{*c.taskArn},
		})
		if err != nil {
			return err
		}
		task := o.Tasks[0]
		if *task.LastStatus != "RUNNING" {
			return xerrors.Errorf("😫 canary task has stopped: %s", *task.StoppedReason)
		}
		return nil
	} else {
		if err := c.registerToTargetGroup(ctx); err != nil {
			return err
		}
		log.Infof("😷 ensuring canary task to become healthy...")
		if err := c.ensureTaskHealthy(ctx); err != nil {
			return err
		}
		log.Info("🤩 canary task is healthy!")
		return nil
	}
}

func (c *CanaryTask) waitUntilHealthCeheckPassed(ctx context.Context) error {
	log.Infof("😷 ensuring canary task container(s) to become healthy...")
	containerHasHealthChecks := map[string]struct{}{}
	for _, definition := range c.td.ContainerDefinitions {
		if definition.HealthCheck != nil {
			containerHasHealthChecks[*definition.Name] = struct{}{}
		}
	}
	for count := 0; count < 10; count++ {
		<-c.Time.NewTimer(time.Duration(15) * time.Second).C
		log.Infof("canary task '%s' waits until %d container(s) become healthy", *c.taskArn, len(containerHasHealthChecks))
		if o, err := c.Ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &c.Env.Cluster,
			Tasks:   []string{*c.taskArn},
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

func (c *CanaryTask) registerToTargetGroup(ctx context.Context) error {
	// Phase 3: Get task details after network interface is attached
	var task ecstypes.Task
	if o, err := c.Ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.Env.Cluster,
		Tasks:   []string{*c.taskArn},
	}); err != nil {
		return err
	} else {
		task = o.Tasks[0]
	}
	var targetId *string
	var targetPort *int32
	var subnet ec2types.Subnet
	for _, container := range c.td.ContainerDefinitions {
		if *container.Name == *c.lb.ContainerName {
			targetPort = container.PortMappings[0].HostPort
		}
	}
	if targetPort == nil {
		return xerrors.Errorf("couldn't find host port in container definition")
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
			return xerrors.Errorf("couldn't find subnetId or privateIPv4Address in task details")
		}
		if o, err := c.Ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: []string{*subnetId},
		}); err != nil {
			return err
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
			return err
		} else {
			containerInstance = outputs.ContainerInstances[0]
		}
		if o, err := c.Ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{*containerInstance.Ec2InstanceId},
		}); err != nil {
			return err
		} else if sn, err := c.Ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: []string{*o.Reservations[0].Instances[0].SubnetId},
		}); err != nil {
			return err
		} else {
			targetId = containerInstance.Ec2InstanceId
			subnet = sn.Subnets[0]
		}
		log.Infof("canary task was placed: instanceId = '%s', hostPort = '%d', az = '%s'", *targetId, *targetPort, *subnet.AvailabilityZone)
	}
	if _, err := c.Alb.RegisterTargets(ctx, &elbv2.RegisterTargetsInput{
		TargetGroupArn: c.lb.TargetGroupArn,
		Targets: []elbv2types.TargetDescription{{
			AvailabilityZone: subnet.AvailabilityZone,
			Id:               targetId,
			Port:             targetPort,
		}},
	}); err != nil {
		return err
	}
	c.target = &CanaryTarget{
		targetGroupArn:   c.lb.TargetGroupArn,
		targetId:         targetId,
		targetPort:       targetPort,
		availabilityZone: subnet.AvailabilityZone,
	}
	return nil
}

func (c *CanaryTask) ensureTaskHealthy(
	ctx context.Context,
) error {
	log.Infof("checking the health state of canary task...")
	var unusedCount = 0
	var initialized = false
	var recentState *elbv2types.TargetHealthStateEnum
	for {
		<-c.Time.NewTimer(time.Duration(15) * time.Second).C
		if o, err := c.Alb.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: c.target.targetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				Id:               c.target.targetId,
				Port:             c.target.targetPort,
				AvailabilityZone: c.target.availabilityZone,
			}},
		}); err != nil {
			return err
		} else {
			for _, desc := range o.TargetHealthDescriptions {
				if *desc.Target.Id == *c.target.targetId && *desc.Target.Port == *c.target.targetPort {
					recentState = &desc.TargetHealth.State
				}
			}
			if recentState == nil {
				return xerrors.Errorf("'%s' is not registered to the target group '%s'", c.target.targetId, c.target.targetGroupArn)
			}
			log.Infof("canary task '%s' (%s:%d) state is: %s", *c.taskArn, c.target.targetId, c.target.targetPort, *recentState)
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
		log.Errorf("😨 canary task '%s' is unhealthy", *c.taskArn)
		return xerrors.Errorf(
			"canary task '%s' (%s:%d) hasn't become to be healthy. The most recent state: %s",
			*c.taskArn, c.target.targetId, c.target.targetPort, *recentState,
		)
	}
}

func (c *CanaryTask) Stop(ctx context.Context) error {
	if c.target == nil {
		log.Info("no load balancer is attached to service. Skip deregisteration.")
	} else {
		log.Infof("deregistering the canary task from target group '%s'...", c.target.targetId)
		if _, err := c.Alb.DeregisterTargets(ctx, &elbv2.DeregisterTargetsInput{
			TargetGroupArn: c.target.targetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				AvailabilityZone: c.target.availabilityZone,
				Id:               c.target.targetId,
				Port:             c.target.targetPort,
			}},
		}); err != nil {
			return err
		}
		if err := elbv2.NewTargetDeregisteredWaiter(c.Alb).Wait(ctx, &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: c.target.targetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				AvailabilityZone: c.target.availabilityZone,
				Id:               c.target.targetId,
				Port:             c.target.targetPort,
			}},
		}, c.MaxWait); err != nil {
			return err
		}
		log.Infof(
			"canary task '%s' has successfully been deregistered from target group '%s'",
			*c.taskArn, c.target.targetId,
		)
	}
	log.Infof("stopping the canary task '%s'...", *c.taskArn)
	if _, err := c.Ecs.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &c.Env.Cluster,
		Task:    c.taskArn,
	}); err != nil {
		return err
	}
	if err := ecs.NewTasksStoppedWaiter(c.Ecs).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.Env.Cluster,
		Tasks:   []string{*c.taskArn},
	}, c.MaxWait); err != nil {
		return err
	}
	log.Infof("canary task '%s' has successfully been stopped", *c.taskArn)
	return nil
}
