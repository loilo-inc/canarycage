package cage

import (
	"context"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type UpResult struct {
	TaskDefinition *ecs.TaskDefinition
	Service        *ecs.Service
}

func (c *cage) Up(ctx context.Context) (*UpResult, error) {
	td, err := c.CreateNextTaskDefinition()
	if err != nil {
		return nil, err
	}
	log.Infof("checking existence of service '%s'", c.env.Service)
	if o, err := c.ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  &c.env.Cluster,
		Services: aws.StringSlice([]string{c.env.Service}),
	}); err != nil {
		log.Fatalf("couldn't describe service: %s", err.Error())
	} else if len(o.Services) > 0 {
		log.Fatalf("service '%s' already exists. Use 'cage rollout' instead", c.env.Service)
	}
	c.env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinitionArn
	log.Infof("creating service '%s' with task-definition '%s'...", c.env.Service, *td.TaskDefinitionArn)
	if o, err := c.ecs.CreateService(c.env.ServiceDefinitionInput); err != nil {
		log.Fatalf("failed to create service '%s': %s", c.env.Service, err.Error())
	} else {
		log.Infof("service created: '%s'", *o.Service.ServiceArn)
	}
	log.Infof("waiting for service '%s' to be STABLE", c.env.Service)
	if err := c.ecs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  &c.env.Cluster,
		Services: []*string{&c.env.Service},
	}); err != nil {
		log.Fatalf(err.Error())
	} else {
		log.Infof("become: STABLE")
	}
	svc, err := c.ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  &c.env.Cluster,
		Services: []*string{&c.env.Service},
	})
	if err != nil {
		return nil, err
	}
	return &UpResult{
		TaskDefinition: td,
		Service:        svc.Services[0],
	}, nil
}
