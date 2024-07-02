package task

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
	"golang.org/x/xerrors"
)

type CanaryTarget struct {
	targetId         string
	targetIpv4       string
	targetPort       int32
	availabilityZone string
}

type Input struct {
	TaskDefinition       *ecstypes.TaskDefinition
	NetworkConfiguration *ecstypes.NetworkConfiguration
	PlatformVersion      *string
}

type common struct {
	*Input
	di      *di.D
	taskArn *string
}

func (c *common) Start(ctx context.Context) error {
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	group := fmt.Sprintf("cage:canary-task:%s", env.Service)
	if env.CanaryInstanceArn != "" {
		// ec2
		if o, err := ecsCli.StartTask(ctx, &ecs.StartTaskInput{
			Cluster:              &env.Cluster,
			Group:                &group,
			NetworkConfiguration: c.NetworkConfiguration,
			TaskDefinition:       c.TaskDefinition.TaskDefinitionArn,
			ContainerInstances:   []string{env.CanaryInstanceArn},
		}); err != nil {
			return err
		} else {
			c.taskArn = o.Tasks[0].TaskArn
		}
	} else {
		// fargate
		if o, err := ecsCli.RunTask(ctx, &ecs.RunTaskInput{
			Cluster:              &env.Cluster,
			Group:                &group,
			NetworkConfiguration: c.NetworkConfiguration,
			TaskDefinition:       c.TaskDefinition.TaskDefinitionArn,
			LaunchType:           ecstypes.LaunchTypeFargate,
			PlatformVersion:      c.PlatformVersion,
		}); err != nil {
			return err
		} else {
			c.taskArn = o.Tasks[0].TaskArn
		}
	}
	return nil
}

func (c *common) TaskArn() *string {
	return c.taskArn
}

func (c *common) waitForTask(ctx context.Context) error {
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	log.Infof("🥚 waiting for canary task '%s' is running...", *c.taskArn)
	if err := ecs.NewTasksRunningWaiter(ecsCli).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*c.taskArn},
	}, env.TaskRunning()); err != nil {
		return err
	}
	log.Infof("🐣 canary task '%s' is running!", *c.taskArn)
	if err := c.waitContainerHealthCheck(ctx); err != nil {
		return err
	}
	log.Info("🤩 canary task container(s) is healthy!")
	log.Infof("canary task '%s' ensured.", *c.taskArn)
	return nil
}

func (c *common) waitContainerHealthCheck(ctx context.Context) error {
	log.Infof("😷 ensuring canary task container(s) to become healthy...")
	env := c.di.Get(key.Env).(*env.Envars)
	timer := c.di.Get(key.Time).(types.Time)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	containerHasHealthChecks := map[string]struct{}{}
	for _, definition := range c.TaskDefinition.ContainerDefinitions {
		if definition.HealthCheck != nil {
			containerHasHealthChecks[*definition.Name] = struct{}{}
		}
	}
	rest := env.TaskHealthCheck()
	healthCheckPeriod := 15 * time.Second
	for rest > 0 {
		if rest < healthCheckPeriod {
			healthCheckPeriod = rest
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.NewTimer(healthCheckPeriod).C:
			log.Infof("canary task '%s' waits until %d container(s) become healthy", *c.taskArn, len(containerHasHealthChecks))
			if o, err := ecsCli.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: &env.Cluster,
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
		rest -= healthCheckPeriod
	}
	return xerrors.Errorf("😨 canary task hasn't become to be healthy")
}

func (c *common) describeTaskTarget(
	ctx context.Context,
	targetPort int32,
) (*CanaryTarget, error) {
	env := c.di.Get(key.Env).(*env.Envars)
	target := CanaryTarget{targetPort: targetPort}
	if env.CanaryInstanceArn == "" { // Fargate
		if err := c.getFargateTarget(ctx, &target); err != nil {
			return nil, err
		}
		log.Infof("canary task was placed: privateIp = '%s', hostPort = '%d', az = '%s'", target.targetId, target.targetPort, target.availabilityZone)
	} else {
		if err := c.getEc2Target(ctx, &target); err != nil {
			return nil, err
		}
		log.Infof("canary task was placed: instanceId = '%s', hostPort = '%d', az = '%s'", target.targetId, target.targetPort, target.availabilityZone)
	}
	return &target, nil
}

func (c *common) getFargateTarget(ctx context.Context, dest *CanaryTarget) error {
	var task ecstypes.Task
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	ec2Cli := c.di.Get(key.Ec2Cli).(awsiface.Ec2Client)
	if o, err := ecsCli.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*c.taskArn},
	}); err != nil {
		return err
	} else {
		task = o.Tasks[0]
	}
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
	if o, err := ec2Cli.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{*subnetId},
	}); err != nil {
		return err
	} else {
		dest.targetId = *privateIp
		dest.targetIpv4 = *privateIp
		dest.availabilityZone = *o.Subnets[0].AvailabilityZone
	}
	return nil
}

func (c *common) getEc2Target(ctx context.Context, dest *CanaryTarget) error {
	var containerInstance ecstypes.ContainerInstance
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	ec2Cli := c.di.Get(key.Ec2Cli).(awsiface.Ec2Client)
	if outputs, err := ecsCli.DescribeContainerInstances(ctx, &ecs.DescribeContainerInstancesInput{
		Cluster:            &env.Cluster,
		ContainerInstances: []string{env.CanaryInstanceArn},
	}); err != nil {
		return err
	} else {
		containerInstance = outputs.ContainerInstances[0]
	}
	var ec2Instance ec2types.Instance
	if o, err := ec2Cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{*containerInstance.Ec2InstanceId},
	}); err != nil {
		return err
	} else {
		ec2Instance = o.Reservations[0].Instances[0]
	}
	if sn, err := ec2Cli.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{*ec2Instance.SubnetId},
	}); err != nil {
		return err
	} else {
		dest.targetId = *containerInstance.Ec2InstanceId
		dest.targetIpv4 = *ec2Instance.PrivateIpAddress
		dest.availabilityZone = *sn.Subnets[0].AvailabilityZone
	}
	return nil
}

func (c *common) stopTask(ctx context.Context) error {
	if c.taskArn == nil {
		return nil
	}
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	log.Infof("stopping the canary task '%s'...", *c.taskArn)
	if _, err := ecsCli.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &env.Cluster,
		Task:    c.taskArn,
	}); err != nil {
		return xerrors.Errorf("failed to stop canary task: %w", err)
	}
	if err := ecs.NewTasksStoppedWaiter(ecsCli).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*c.taskArn},
	}, env.TaskStopped()); err != nil {
		return xerrors.Errorf("failed to wait for canary task to be stopped: %w", err)
	}
	log.Infof("canary task '%s' has successfully been stopped", *c.taskArn)
	return nil
}
