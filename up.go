package cage

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
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
	log.Infof("checking existence of service '%s'", c.env.Service)
	if o, err := c.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.env.Cluster,
		Services: []string{c.env.Service},
	}); err != nil {
		return nil, fmt.Errorf("couldn't describe service: %s", err.Error())
	} else if len(o.Services) > 0 {
		svc := o.Services[0]
		if *svc.Status != "INACTIVE" {
			return nil, fmt.Errorf("service '%s' already exists. Use 'cage rollout' instead", c.env.Service)
		}
	}
	c.env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinitionArn
	if service, err := c.createService(ctx, c.env.ServiceDefinitionInput); err != nil {
		return nil, err
	} else {
		return &UpResult{TaskDefinition: td, Service: service}, nil
	}
}
