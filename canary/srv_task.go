package canary

import (
	"context"
	"regexp"
	"time"

	"github.com/apex/log"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	srvtypes "github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"golang.org/x/xerrors"
)

type srvTask struct {
	*common
	registry *ecstypes.ServiceRegistry
	target   *CanaryTarget
	srv      *srvtypes.Service
	inst     *srvtypes.HttpInstanceSummary
}

func NewSrvTask(input *Input, registry *ecstypes.ServiceRegistry) Task {
	return &srvTask{
		common:   &common{Input: input},
		registry: registry,
	}
}

func (c *srvTask) Wait(ctx context.Context) error {
	if err := c.wait(ctx); err != nil {
		return err
	}
	if err := c.registerToSrvDiscovery(ctx); err != nil {
		return err
	}
	log.Infof("😷 ensuring canary task to become healthy...")
	if err := c.waitUntilSrvInstHelthy(ctx); err != nil {
		return err
	}
	log.Info("🤩 canary task is healthy!")
	return nil
}

func (c *srvTask) Stop(ctx context.Context) error {
	if err := c.deregisterSrvInst(ctx); err != nil {
		return err
	}
	return c.stopTask(ctx)
}

func (c *srvTask) getTargetPort(ctx context.Context) (int32, error) {
	if c.registry.Port != nil {
		return *c.registry.Port, nil
	}
	return 80, nil
}

func (c *srvTask) registerToSrvDiscovery(ctx context.Context) error {
	target, err := c.describeTaskTarget(ctx, c.getTargetPort)
	if err != nil {
		return err
	}
	c.target = target
	// get the service id from service registry arn
	pat := regexp.MustCompile("arn://.+/(srv-.+)$")
	matches := pat.FindStringSubmatch(*c.registry.RegistryArn)
	if len(matches) != 2 {
		return xerrors.Errorf("service name '%s' doesn't match the pattern", c.Env.Service)
	}
	srvId := matches[1]
	var svc *srvtypes.Service
	if o, err := c.Srv.GetService(ctx, &servicediscovery.GetServiceInput{
		Id: &srvId,
	}); err != nil {
		return xerrors.Errorf("failed to get the service: %w", err)
	} else {
		svc = o.Service
	}
	attrs := map[string]string{
		"AWS_INSTANCE_IPV4":          target.targetIpv4,
		"AVAILABILITY_ZONE":          target.availabilityZone,
		"AWS_INIT_HEALTH_STATUS":     "UNHEALTHY",
		"ECS_CLUSTER_NAME":           c.Env.Cluster,
		"ECS_SERVICE_NAME":           c.Env.Service,
		"ECS_TASK_DEFINITION_FAMILY": *c.TaskDefinition.Family,
		"REGION":                     c.Env.Region,
		"CAGE_CANARY_TASK":           "1",
	}
	if _, err := c.Srv.RegisterInstance(ctx, &servicediscovery.RegisterInstanceInput{
		ServiceId:  &srvId,
		InstanceId: c.taskArn,
		Attributes: attrs,
	}); err != nil {
		return xerrors.Errorf("failed to register the canary task to service discovery: %w", err)
	}
	c.srv = svc
	return nil
}

func (c *srvTask) waitUntilSrvInstHelthy(
	ctx context.Context,
) error {
	var maxWait = 900
	var waitPeriod = 15
	for maxWait > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.Time.NewTimer(time.Duration(waitPeriod) * time.Second).C:
			if list, err := c.Srv.DiscoverInstances(ctx, &servicediscovery.DiscoverInstancesInput{
				NamespaceName: c.inst.NamespaceName,
				ServiceName:   c.inst.ServiceName,
				HealthStatus:  srvtypes.HealthStatusFilterHealthy,
				QueryParameters: map[string]string{
					"CAGE_CANARY_TASK": "1",
				},
			}); err != nil {
				return xerrors.Errorf("failed to discover instances: %w", err)
			} else {
				if len(list.Instances) == 0 {
					return xerrors.Errorf("no healthy instances found")
				}
				for _, inst := range list.Instances {
					if ipv4 := inst.Attributes["AWS_INSTANCE_IPV4"]; ipv4 == c.target.targetIpv4 {
						c.inst = &inst
						return nil
					}
				}
				maxWait -= waitPeriod
			}
		}
	}
	return xerrors.Errorf("timed out waiting for healthy instances")
}

func (c *srvTask) deregisterSrvInst(
	ctx context.Context,
) error {
	if _, err := c.Srv.DeregisterInstance(ctx, &servicediscovery.DeregisterInstanceInput{
		ServiceId:  c.srv.Id,
		InstanceId: c.inst.InstanceId,
	}); err != nil {
		return xerrors.Errorf("failed to deregister the canary task from service discovery: %w", err)
	}
	return nil
}
