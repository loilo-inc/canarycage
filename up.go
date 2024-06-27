package cage

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/types"
	"golang.org/x/xerrors"
)

func (c *cage) Up(ctx context.Context) (*types.UpResult, error) {
	td, err := c.CreateNextTaskDefinition(ctx)
	if err != nil {
		return nil, err
	}
	log.Infof("checking existence of service '%s'", c.Env.Service)
	if o, err := c.Ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.Env.Cluster,
		Services: []string{c.Env.Service},
	}); err != nil {
		return nil, xerrors.Errorf("couldn't describe service: %w", err)
	} else if len(o.Services) > 0 {
		svc := o.Services[0]
		if *svc.Status != "INACTIVE" {
			return nil, xerrors.Errorf("service '%s' already exists. Use 'cage rollout' instead", c.Env.Service)
		}
	}
	c.Env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinitionArn
	if service, err := c.createService(ctx, c.Env.ServiceDefinitionInput); err != nil {
		return nil, err
	} else {
		return &types.UpResult{TaskDefinition: td, Service: service}, nil
	}
}

func (c *cage) createService(ctx context.Context, serviceDefinitionInput *ecs.CreateServiceInput) (*ecstypes.Service, error) {
	log.Infof("creating service '%s' with task-definition '%s'...", *serviceDefinitionInput.ServiceName, *serviceDefinitionInput.TaskDefinition)
	o, err := c.Ecs.CreateService(ctx, serviceDefinitionInput)
	if err != nil {
		return nil, xerrors.Errorf("failed to create service '%s': %w", *serviceDefinitionInput.ServiceName, err)
	}
	log.Infof("waiting for service '%s' to be STABLE", *serviceDefinitionInput.ServiceName)
	if err := ecs.NewServicesStableWaiter(c.Ecs).Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.Env.Cluster,
		Services: []string{*serviceDefinitionInput.ServiceName},
	}, c.Timeout.ServiceStable()); err != nil {
		return nil, xerrors.Errorf("failed to wait for service '%s' to be STABLE: %w", *serviceDefinitionInput.ServiceName, err)
	}
	return o.Service, nil
}
