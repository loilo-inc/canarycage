package task

import (
	"context"
	"time"

	"github.com/apex/log"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	srvtypes "github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
	"golang.org/x/xerrors"
)

// srvTask is a task that is attached to an Service Discovery
type srvTask struct {
	*common
	registry *ecstypes.ServiceRegistry
	target   *CanaryTarget
	srv      *srvtypes.Service
	instId   *string
	ns       *srvtypes.Namespace
}

func NewSrvTask(di *di.D, input *Input, registry *ecstypes.ServiceRegistry) Task {
	return &srvTask{
		common:   &common{Input: input, di: di},
		registry: registry,
	}
}

func (c *srvTask) Wait(ctx context.Context) error {
	if err := c.waitForTask(ctx); err != nil {
		return err
	}
	if err := c.registerToSrvDiscovery(ctx); err != nil {
		return err
	}
	log.Infof("canary task '%s' is registered to service discovery instance '%s'", *c.taskArn, *c.instId)
	log.Infof("😷 ensuring canary service instance to become healthy...")
	if err := c.waitUntilSrvInstHelthy(ctx); err != nil {
		return err
	}
	log.Info("🤩 canary service instance is healthy!")
	return nil
}

func (c *srvTask) Stop(ctx context.Context) error {
	c.deregisterSrvInst(ctx)
	return c.stopTask(ctx)
}

func (c *srvTask) registerToSrvDiscovery(ctx context.Context) error {
	log.Infof("registring canary task '%s' to service discovery...", *c.taskArn)
	var targetPort int32
	if c.registry.Port != nil {
		targetPort = *c.registry.Port
	} else {
		targetPort = 80
	}
	target, err := c.describeTaskTarget(ctx, targetPort)
	if err != nil {
		return err
	}
	c.target = target // get the service id from service registry arn
	srvId := ArnToId(*c.registry.RegistryArn)
	srvCli := c.di.Get(key.SrvCli).(awsiface.SrvClient)
	env := c.di.Get(key.Env).(*env.Envars)
	var svc *srvtypes.Service
	var ns *srvtypes.Namespace
	if o, err := srvCli.GetService(ctx, &servicediscovery.GetServiceInput{
		Id: &srvId,
	}); err != nil {
		return xerrors.Errorf("failed to get the service: %w", err)
	} else {
		svc = o.Service
	}
	if o, err := srvCli.GetNamespace(ctx, &servicediscovery.GetNamespaceInput{
		Id: svc.NamespaceId,
	}); err != nil {
		return xerrors.Errorf("failed to get the namespace: %w", err)
	} else {
		ns = o.Namespace
	}
	attrs := map[string]string{
		"AWS_INSTANCE_IPV4":          target.targetIpv4,
		"AVAILABILITY_ZONE":          target.availabilityZone,
		"AWS_INIT_HEALTH_STATUS":     "UNHEALTHY",
		"ECS_CLUSTER_NAME":           env.Cluster,
		"ECS_SERVICE_NAME":           env.Service,
		"ECS_TASK_DEFINITION_FAMILY": *c.TaskDefinition.Family,
		"REGION":                     env.Region,
		"CAGE_TASK_ID":               ArnToId(*c.taskArn),
	}
	taskId := ArnToId(*c.taskArn)
	if _, err := srvCli.RegisterInstance(ctx, &servicediscovery.RegisterInstanceInput{
		ServiceId:  &srvId,
		InstanceId: &taskId,
		Attributes: attrs,
	}); err != nil {
		return xerrors.Errorf("failed to register the canary task to service discovery: %w", err)
	}
	c.srv = svc
	c.instId = &taskId
	c.ns = ns
	return nil
}

func (c *srvTask) waitUntilSrvInstHelthy(
	ctx context.Context,
) error {
	env := c.di.Get(key.Env).(*env.Envars)
	timer := c.di.Get(key.Time).(types.Time)
	srvCli := c.di.Get(key.SrvCli).(awsiface.SrvClient)
	var rest = env.TargetHealthCheck()
	var waitPeriod = 15 * time.Second
	for rest > 0 {
		if rest < waitPeriod {
			waitPeriod = rest
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.NewTimer(time.Duration(waitPeriod) * time.Second).C:
			if list, err := srvCli.DiscoverInstances(ctx, &servicediscovery.DiscoverInstancesInput{
				NamespaceName: c.ns.Name,
				ServiceName:   c.srv.Name,
				HealthStatus:  srvtypes.HealthStatusFilterHealthy,
				QueryParameters: map[string]string{
					"CAGE_TASK_ID": ArnToId(*c.taskArn),
				},
			}); err != nil {
				return xerrors.Errorf("failed to discover instances: %w", err)
			} else {
				for _, inst := range list.Instances {
					if *inst.InstanceId == *c.instId {
						return nil
					}
				}
				rest -= waitPeriod
			}
		}
	}
	return xerrors.Errorf("timed out waiting for healthy instances")
}

func (c *srvTask) deregisterSrvInst(
	ctx context.Context,
) {
	if c.instId == nil {
		return
	}
	log.Info("deregistering the canary task from service discovery...")
	srvCli := c.di.Get(key.SrvCli).(awsiface.SrvClient)
	if _, err := srvCli.DeregisterInstance(ctx, &servicediscovery.DeregisterInstanceInput{
		ServiceId:  c.srv.Id,
		InstanceId: c.instId,
	}); err != nil {
		log.Errorf("failed to deregister the canary task from service discovery: %v", err)
		log.Errorf("continuing to stop the canary task...")
	}
	log.Infof("canary task '%s' is deregistered from service discovery", *c.taskArn)
}
