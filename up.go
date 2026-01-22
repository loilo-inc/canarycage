package cage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/canarycage/types"
	"golang.org/x/xerrors"
)

func (c *cage) Up(ctx context.Context) (*types.UpResult, error) {
	td, err := c.CreateNextTaskDefinition(ctx)
	if err != nil {
		return nil, err
	}
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	log := c.di.Get(key.Logger).(logger.Logger)
	log.Infof("checking existence of service '%s'", env.Service)
	if o, err := ecsCli.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &env.Cluster,
		Services: []string{env.Service},
	}); err != nil {
		return nil, xerrors.Errorf("couldn't describe service: %w", err)
	} else if len(o.Services) > 0 {
		svc := o.Services[0]
		if *svc.Status != "INACTIVE" {
			return nil, xerrors.Errorf("service '%s' already exists. Use 'cage rollout' instead", env.Service)
		}
	}
	env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinitionArn
	if service, err := c.createService(ctx, env.ServiceDefinitionInput); err != nil {
		return nil, err
	} else {
		return &types.UpResult{TaskDefinition: td, Service: service}, nil
	}
}

func (c *cage) createService(ctx context.Context, serviceDefinitionInput *ecs.CreateServiceInput) (*ecstypes.Service, error) {
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	log := c.di.Get(key.Logger).(logger.Logger)
	log.Infof("creating service '%s' with task-definition '%s'...", *serviceDefinitionInput.ServiceName, *serviceDefinitionInput.TaskDefinition)
	o, err := ecsCli.CreateService(ctx, serviceDefinitionInput)
	if err != nil {
		return nil, xerrors.Errorf("failed to create service '%s': %w", *serviceDefinitionInput.ServiceName, err)
	}
	log.Infof("waiting for service '%s' to be STABLE", *serviceDefinitionInput.ServiceName)
	if err := ecs.NewServicesStableWaiter(ecsCli).Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &env.Cluster,
		Services: []string{*serviceDefinitionInput.ServiceName},
	}, env.GetServiceStableWait()); err != nil {
		return nil, xerrors.Errorf("failed to wait for service '%s' to be STABLE: %w", *serviceDefinitionInput.ServiceName, err)
	}
	return o.Service, nil
}
