package cage

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"golang.org/x/xerrors"
)

type UpResult struct {
	TaskDefinition *ecstypes.TaskDefinition
	Service        *ecstypes.Service
}

func (c *cage) Up(ctx context.Context) (*UpResult, error) {
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
		return &UpResult{TaskDefinition: td, Service: service}, nil
	}
}
