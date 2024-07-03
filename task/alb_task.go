package task

import (
	"context"
	"strconv"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
	"golang.org/x/xerrors"
)

// albTask is a task that is attached to an Application Load Balancer
type albTask struct {
	*common
	Lb     *ecstypes.LoadBalancer
	Target *CanaryTarget
}

func NewAlbTask(
	di *di.D,
	input *Input,
	lb *ecstypes.LoadBalancer,
) Task {
	return &albTask{
		common: &common{Input: input, di: di},
		Lb:     lb,
	}
}

func (c *albTask) Wait(ctx context.Context) error {
	if err := c.waitForTask(ctx); err != nil {
		return err
	}
	if err := c.registerToTargetGroup(ctx); err != nil {
		return err
	}
	log.Infof("canary task '%s' is registered to target group '%s'", c.Target.targetId, *c.Lb.TargetGroupArn)
	log.Infof("😷 waiting canary target to be healthy...")
	if err := c.waitUntilTargetHealthy(ctx); err != nil {
		return err
	}
	log.Info("🤩 canary target is healthy!")
	return nil
}

func (c *albTask) Stop(ctx context.Context) error {
	c.deregisterTarget(ctx)
	return c.stopTask(ctx)
}

func (c *albTask) describeTaskTarget(
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

func (c *albTask) getFargateTarget(ctx context.Context, dest *CanaryTarget) error {
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

func (c *albTask) getEc2Target(ctx context.Context, dest *CanaryTarget) error {
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

func (c *albTask) getTargetPort() (int32, error) {
	for _, container := range c.TaskDefinition.ContainerDefinitions {
		if *container.Name == *c.Lb.ContainerName {
			return *container.PortMappings[0].HostPort, nil
		}
	}
	return 0, xerrors.Errorf("couldn't find host port in container definition")
}

func (c *albTask) registerToTargetGroup(ctx context.Context) error {
	log.Infof("registering the canary task to target group '%s'...", *c.Lb.TargetGroupArn)
	if targetPort, err := c.getTargetPort(); err != nil {
		return err
	} else if target, err := c.describeTaskTarget(ctx, targetPort); err != nil {
		return err
	} else {
		c.Target = target
	}
	albCli := c.di.Get(key.AlbCli).(awsiface.AlbClient)
	if _, err := albCli.RegisterTargets(ctx, &elbv2.RegisterTargetsInput{
		TargetGroupArn: c.Lb.TargetGroupArn,
		Targets: []elbv2types.TargetDescription{{
			AvailabilityZone: &c.Target.availabilityZone,
			Id:               &c.Target.targetId,
			Port:             &c.Target.targetPort,
		}},
	}); err != nil {
		return err
	}
	return nil
}

func (c *albTask) waitUntilTargetHealthy(
	ctx context.Context,
) error {
	albCli := c.di.Get(key.AlbCli).(awsiface.AlbClient)
	timer := c.di.Get(key.Time).(types.Time)
	log.Infof("checking the health state of canary task...")
	var notHealthyCount = 0
	var recentState *elbv2types.TargetHealthStateEnum
	waitPeriod := 15 * time.Second
	for notHealthyCount < 5 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.NewTimer(waitPeriod).C:
			if o, err := albCli.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
				TargetGroupArn: c.Lb.TargetGroupArn,
				Targets: []elbv2types.TargetDescription{{
					Id:               &c.Target.targetId,
					Port:             &c.Target.targetPort,
					AvailabilityZone: &c.Target.availabilityZone,
				}},
			}); err != nil {
				return err
			} else {
				for _, desc := range o.TargetHealthDescriptions {
					if *desc.Target.Id == c.Target.targetId && *desc.Target.Port == c.Target.targetPort {
						recentState = &desc.TargetHealth.State
					}
				}
				if recentState == nil {
					return xerrors.Errorf("'%s' is not registered to the target group '%s'", c.Target.targetId, *c.Lb.TargetGroupArn)
				}
				log.Infof("canary task '%s' (%s:%d) state is: %s", *c.taskArn, c.Target.targetId, c.Target.targetPort, *recentState)
				switch *recentState {
				case elbv2types.TargetHealthStateEnumHealthy:
					return nil
				default:
					notHealthyCount++
				}
			}
		}
	}
	// unhealthy, draining, unused
	log.Errorf("😨 canary task '%s' is unhealthy", *c.taskArn)
	return xerrors.Errorf(
		"canary task '%s' (%s:%d) hasn't become to be healthy. The most recent state: %s",
		*c.taskArn, c.Target.targetId, c.Target.targetPort, *recentState,
	)
}

func (c *albTask) targetDeregistrationDelay(ctx context.Context) (time.Duration, error) {
	deregistrationDelay := 300 * time.Second
	albCli := c.di.Get(key.AlbCli).(awsiface.AlbClient)
	if o, err := albCli.DescribeTargetGroupAttributes(ctx, &elbv2.DescribeTargetGroupAttributesInput{
		TargetGroupArn: c.Lb.TargetGroupArn,
	}); err != nil {
		return deregistrationDelay, err
	} else {
		// find deregistration_delay.timeout_seconds
		for _, attr := range o.Attributes {
			if *attr.Key == "deregistration_delay.timeout_seconds" {
				if value, err := strconv.ParseInt(*attr.Value, 10, 64); err != nil {
					return deregistrationDelay, err
				} else {
					deregistrationDelay = time.Duration(value) * time.Second
				}
			}
		}
	}
	return deregistrationDelay, nil
}

func (c *albTask) deregisterTarget(ctx context.Context) {
	if c.Target == nil {
		return
	}
	albCli := c.di.Get(key.AlbCli).(awsiface.AlbClient)
	deregistrationDelay, err := c.targetDeregistrationDelay(ctx)
	if err != nil {
		log.Errorf("failed to get deregistration delay: %v", err)
		log.Errorf("deregistration delay is set to %d seconds", deregistrationDelay)
	}
	log.Infof("deregistering the canary task from target group '%s'...", c.Target.targetId)
	if _, err := albCli.DeregisterTargets(ctx, &elbv2.DeregisterTargetsInput{
		TargetGroupArn: c.Lb.TargetGroupArn,
		Targets: []elbv2types.TargetDescription{{
			AvailabilityZone: &c.Target.availabilityZone,
			Id:               &c.Target.targetId,
			Port:             &c.Target.targetPort,
		}},
	}); err != nil {
		log.Errorf("failed to deregister the canary task from target group: %v", err)
		log.Errorf("continuing to stop the canary task...")
	} else {
		log.Infof("deregister operation accepted. waiting for the canary task to be deregistered...")
		deregisterWait := deregistrationDelay + time.Minute // add 1 minute for safety
		if err := elbv2.NewTargetDeregisteredWaiter(albCli).Wait(ctx, &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: c.Lb.TargetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				AvailabilityZone: &c.Target.availabilityZone,
				Id:               &c.Target.targetId,
				Port:             &c.Target.targetPort,
			}},
		}, deregisterWait); err != nil {
			log.Errorf("failed to wait for the canary task deregistered from target group: %v", err)
			log.Errorf("continuing to stop the canary task...")
		} else {
			log.Infof(
				"canary task '%s' has successfully been deregistered from target group '%s'",
				*c.taskArn, c.Target.targetId,
			)
		}
	}
}
