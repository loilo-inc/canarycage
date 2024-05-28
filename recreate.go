package cage

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"golang.org/x/xerrors"
)

type RecreateResult struct {
	Service        *ecstypes.Service
	TaskDefinition *ecstypes.TaskDefinition
}

func (c *cage) Recreate(ctx context.Context) (*RecreateResult, error) {
	// Check if the service already exists
	log.Infof("checking existence of service '%s'", c.Env.Service)
	var oldService *ecstypes.Service
	var transitService *ecstypes.Service
	var newService *ecstypes.Service
	if o, err := c.Ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.Env.Cluster,
		Services: []string{c.Env.Service},
	}); err != nil {
		return nil, xerrors.Errorf("couldn't describe service: %w", err)
	} else if len(o.Services) == 0 {
		return nil, fmt.Errorf("service '%s' does not exist. Use 'cage up' instead", c.Env.Service)
	} else {
		oldService = &o.Services[0]
		if *oldService.Status == "INACTIVE" {
			return nil, fmt.Errorf("service '%s' is already INACTIVE. Use 'cage up' instead", c.Env.Service)
		}
	}
	var err error
	// Create a new task definition
	td, err := c.CreateNextTaskDefinition(ctx)
	if err != nil {
		return nil, err
	}
	transitServiceName := fmt.Sprintf("%s-%d", *oldService.ServiceName, c.Time.Now().Unix())
	newServiceInput := *c.Env.ServiceDefinitionInput
	curDesiredCount := oldService.DesiredCount
	c.Env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinitionArn
	transitServiceDifinitonInput := *c.Env.ServiceDefinitionInput
	transitServiceDifinitonInput.ServiceName = &transitServiceName
	transitServiceDifinitonInput.DesiredCount = aws.Int32(1)
	// Create a transit service
	if transitService, err = c.createService(ctx, &transitServiceDifinitonInput); err != nil {
		return nil, err
	}
	// Update transit service to same task count as previous service
	if err = c.updateServiceTaskCount(ctx, *transitService.ServiceName, oldService.DesiredCount); err != nil {
		return nil, err
	}
	// Update old service to 0 tasks
	if err = c.updateServiceTaskCount(ctx, *oldService.ServiceName, 0); err != nil {
		return nil, err
	}
	// Delete old service
	if err = c.deleteService(ctx, *oldService.ServiceName); err != nil {
		return nil, err
	}
	oldService = nil
	// Create a new service
	if newService, err = c.createService(ctx, &newServiceInput); err != nil {
		return nil, err
	}
	// Update new service to same task count as transit service
	if err = c.updateServiceTaskCount(ctx, *newService.ServiceName, curDesiredCount); err != nil {
		return nil, err
	}
	// Update transit service to 0 tasks
	if err = c.updateServiceTaskCount(ctx, *transitService.ServiceName, 0); err != nil {
		return nil, err
	}
	// Delete transit service
	if err = c.deleteService(ctx, *transitService.ServiceName); err != nil {
		return nil, err
	}
	transitService = nil
	return &RecreateResult{TaskDefinition: td, Service: newService}, nil
}

func (c *cage) createService(ctx context.Context, serviceDefinitionInput *ecs.CreateServiceInput) (*ecstypes.Service, error) {
	log.Infof("creating service '%s' with task-definition '%s'...", *serviceDefinitionInput.ServiceName, *serviceDefinitionInput.TaskDefinition)
	o, err := c.Ecs.CreateService(ctx, serviceDefinitionInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create service '%s': %s", *serviceDefinitionInput.ServiceName, err.Error())
	}
	log.Infof("waiting for service '%s' to be STABLE", *serviceDefinitionInput.ServiceName)
	if err := ecs.NewServicesStableWaiter(c.Ecs).Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.Env.Cluster,
		Services: []string{*serviceDefinitionInput.ServiceName},
	}, WaitDuration); err != nil {
		return nil, fmt.Errorf("failed to wait for service '%s' to be STABLE: %s", *serviceDefinitionInput.ServiceName, err.Error())
	}
	return o.Service, nil
}

func (c *cage) updateServiceTaskCount(ctx context.Context, service string, count int32) error {
	log.Infof("updating service '%s' desired count to %d...", service, count)
	if _, err := c.Ecs.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:      &c.Env.Cluster,
		Service:      &service,
		DesiredCount: &count,
	}); err != nil {
		return fmt.Errorf("failed to update service '%s': %w", service, err)
	}
	log.Infof("waiting for service '%s' to be STABLE", service)
	if err := ecs.NewServicesStableWaiter(c.Ecs).Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.Env.Cluster,
		Services: []string{service},
	}, WaitDuration); err != nil {
		return fmt.Errorf("failed to wait for service '%s' to be STABLE: %v", service, err)
	}
	return nil
}

func (c *cage) deleteService(ctx context.Context, service string) error {
	log.Infof("deleting service '%s'...", service)
	if _, err := c.Ecs.DeleteService(ctx, &ecs.DeleteServiceInput{
		Cluster: &c.Env.Cluster,
		Service: &service,
	}); err != nil {
		return fmt.Errorf("failed to delete service '%s': %w", service, err)
	}
	var retryCount int = 0
	for retryCount < 10 {
		<-c.Time.NewTimer(15 * time.Second).C
		log.Infof("waiting for service '%s' to be INACTIVE", service)
		if o, err := c.Ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  &c.Env.Cluster,
			Services: []string{service},
		}); err != nil {
			return fmt.Errorf("failed to describe service '%s': %w", service, err)
		} else if len(o.Services) == 0 {
			break
		} else if *o.Services[0].Status == "INACTIVE" {
			break
		}
		retryCount++
	}
	return nil
}
