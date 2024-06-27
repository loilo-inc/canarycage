package task

import (
	"context"
	"strconv"
	"time"

	"github.com/apex/log"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"golang.org/x/xerrors"
)

// albTask is a task that is attached to an Application Load Balancer
type albTask struct {
	*common
	lb     *ecstypes.LoadBalancer
	target *CanaryTarget
}

func NewAlbTask(input *Input,
	lb *ecstypes.LoadBalancer,
) Task {
	return &albTask{
		common: &common{Input: input},
		lb:     lb,
	}
}

func (c *albTask) Wait(ctx context.Context) error {
	if err := c.waitForTask(ctx); err != nil {
		return err
	}
	if err := c.registerToTargetGroup(ctx); err != nil {
		return err
	}
	log.Infof("canary task '%s' is registered to target group '%s'", c.target.targetId, *c.lb.TargetGroupArn)
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

func (c *albTask) getTargetPort() (int32, error) {
	for _, container := range c.TaskDefinition.ContainerDefinitions {
		if *container.Name == *c.lb.ContainerName {
			return *container.PortMappings[0].HostPort, nil
		}
	}
	return 0, xerrors.Errorf("couldn't find host port in container definition")
}

func (c *albTask) registerToTargetGroup(ctx context.Context) error {
	log.Infof("registering the canary task to target group '%s'...", *c.lb.TargetGroupArn)
	if targetPort, err := c.getTargetPort(); err != nil {
		return err
	} else if target, err := c.describeTaskTarget(ctx, targetPort); err != nil {
		return err
	} else {
		c.target = target
	}
	if _, err := c.Alb.RegisterTargets(ctx, &elbv2.RegisterTargetsInput{
		TargetGroupArn: c.lb.TargetGroupArn,
		Targets: []elbv2types.TargetDescription{{
			AvailabilityZone: &c.target.availabilityZone,
			Id:               &c.target.targetId,
			Port:             &c.target.targetPort,
		}},
	}); err != nil {
		return err
	}
	return nil
}

func (c *albTask) waitUntilTargetHealthy(
	ctx context.Context,
) error {
	log.Infof("checking the health state of canary task...")
	var unusedCount = 0
	var recentState *elbv2types.TargetHealthStateEnum
	rest := c.Timeout.TargetHealthCheck()
	waitPeriod := 15 * time.Second
	for rest > 0 && unusedCount < 5 {
		if rest < waitPeriod {
			waitPeriod = rest
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.Time.NewTimer(waitPeriod).C:
			if o, err := c.Alb.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
				TargetGroupArn: c.lb.TargetGroupArn,
				Targets: []elbv2types.TargetDescription{{
					Id:               &c.target.targetId,
					Port:             &c.target.targetPort,
					AvailabilityZone: &c.target.availabilityZone,
				}},
			}); err != nil {
				return err
			} else {
				for _, desc := range o.TargetHealthDescriptions {
					if *desc.Target.Id == c.target.targetId && *desc.Target.Port == c.target.targetPort {
						recentState = &desc.TargetHealth.State
					}
				}
				if recentState == nil {
					return xerrors.Errorf("'%s' is not registered to the target group '%s'", c.target.targetId, *c.lb.TargetGroupArn)
				}
				log.Infof("canary task '%s' (%s:%d) state is: %s", *c.taskArn, c.target.targetId, c.target.targetPort, *recentState)
				switch *recentState {
				case "healthy":
					return nil
				case "unused":
					unusedCount++
				}
			}
		}
		rest -= waitPeriod
	}
	// unhealthy, draining, unused
	log.Errorf("😨 canary task '%s' is unhealthy", *c.taskArn)
	return xerrors.Errorf(
		"canary task '%s' (%s:%d) hasn't become to be healthy. The most recent state: %s",
		*c.taskArn, c.target.targetId, c.target.targetPort, *recentState,
	)
}

func (c *albTask) targetDeregistrationDelay(ctx context.Context) (time.Duration, error) {
	deregistrationDelay := 300 * time.Second
	if o, err := c.Alb.DescribeTargetGroupAttributes(ctx, &elbv2.DescribeTargetGroupAttributesInput{
		TargetGroupArn: c.lb.TargetGroupArn,
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
	if c.target == nil {
		return
	}
	deregistrationDelay, err := c.targetDeregistrationDelay(ctx)
	if err != nil {
		log.Errorf("failed to get deregistration delay: %v", err)
		log.Errorf("deregistration delay is set to %d seconds", deregistrationDelay)
	}
	log.Infof("deregistering the canary task from target group '%s'...", c.target.targetId)
	if _, err := c.Alb.DeregisterTargets(ctx, &elbv2.DeregisterTargetsInput{
		TargetGroupArn: c.lb.TargetGroupArn,
		Targets: []elbv2types.TargetDescription{{
			AvailabilityZone: &c.target.availabilityZone,
			Id:               &c.target.targetId,
			Port:             &c.target.targetPort,
		}},
	}); err != nil {
		log.Errorf("failed to deregister the canary task from target group: %v", err)
		log.Errorf("continuing to stop the canary task...")
	} else {
		log.Infof("deregister operation accepted. waiting for the canary task to be deregistered...")
		deregisterWait := deregistrationDelay + time.Minute // add 1 minute for safety
		if err := elbv2.NewTargetDeregisteredWaiter(c.Alb).Wait(ctx, &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: c.lb.TargetGroupArn,
			Targets: []elbv2types.TargetDescription{{
				AvailabilityZone: &c.target.availabilityZone,
				Id:               &c.target.targetId,
				Port:             &c.target.targetPort,
			}},
		}, deregisterWait); err != nil {
			log.Errorf("failed to wait for the canary task deregistered from target group: %v", err)
			log.Errorf("continuing to stop the canary task...")
		} else {
			log.Infof(
				"canary task '%s' has successfully been deregistered from target group '%s'",
				*c.taskArn, c.target.targetId,
			)
		}
	}
}
