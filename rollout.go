package cage

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/types"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

func (c *cage) RollOut(ctx context.Context, input *types.RollOutInput) (*types.RollOutResult, error) {
	result := &types.RollOutResult{
		ServiceIntact: true,
	}
	if out, err := c.Ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster: &c.Env.Cluster,
		Services: []string{
			c.Env.Service,
		},
	}); err != nil {
		return result, xerrors.Errorf("failed to describe current service due to: %w", err)
	} else if len(out.Services) == 0 {
		return result, xerrors.Errorf("service '%s' doesn't exist. Run 'cage up' or create service before rolling out", c.Env.Service)
	} else {
		service := out.Services[0]
		if *service.Status != "ACTIVE" {
			return result, xerrors.Errorf("😵 '%s' status is '%s'. Stop rolling out", c.Env.Service, *service.Status)
		}
		if service.LaunchType == ecstypes.LaunchTypeEc2 && c.Env.CanaryInstanceArn == "" {
			return result, xerrors.Errorf("🥺 --canaryInstanceArn is required when LaunchType = 'EC2'")
		}
	}
	log.Infof("ensuring next task definition...")
	var nextTaskDefinition *ecstypes.TaskDefinition
	if o, err := c.CreateNextTaskDefinition(ctx); err != nil {
		return result, xerrors.Errorf("failed to register next task definition due to: %w", err)
	} else {
		nextTaskDefinition = o
	}
	if input.UpdateService {
		log.Info("--updateService flag is set. use provided service configurations for canary test instead of current service")
	}
	log.Infof("starting canary task...")
	canaryTasks, startCanaryTaskErr := c.StartCanaryTasks(ctx, nextTaskDefinition, input)
	// ensure canary task stopped after rolling out either success or failure
	defer func() {
		_ = recover()
		eg := errgroup.Group{}
		for _, canaryTask := range canaryTasks {
			eg.Go(func() error {
				return canaryTask.Stop(ctx)
			})
		}
		if err := eg.Wait(); err != nil {
			log.Errorf("failed to stop canary tasks due to: %s", err)
		}
	}()
	if startCanaryTaskErr != nil {
		return result, xerrors.Errorf("failed to start canary task due to: %w", startCanaryTaskErr)
	}
	eg := errgroup.Group{}
	for _, canaryTask := range canaryTasks {
		eg.Go(func() error {
			return canaryTask.Wait(ctx)
		})
	}
	if err := eg.Wait(); err != nil {
		return result, xerrors.Errorf("failed to wait for canary task due to: %w", err)
	}
	log.Infof(
		"updating the task definition of '%s' into '%s:%d'...",
		c.Env.Service, *nextTaskDefinition.Family, nextTaskDefinition.Revision,
	)
	updateInput := &ecs.UpdateServiceInput{
		Cluster:        &c.Env.Cluster,
		Service:        &c.Env.Service,
		TaskDefinition: nextTaskDefinition.TaskDefinitionArn,
	}
	if input.UpdateService {
		updateInput.LoadBalancers = c.Env.ServiceDefinitionInput.LoadBalancers
		updateInput.NetworkConfiguration = c.Env.ServiceDefinitionInput.NetworkConfiguration
		updateInput.ServiceConnectConfiguration = c.Env.ServiceDefinitionInput.ServiceConnectConfiguration
		updateInput.ServiceRegistries = c.Env.ServiceDefinitionInput.ServiceRegistries
		updateInput.PlatformVersion = c.Env.ServiceDefinitionInput.PlatformVersion
		updateInput.VolumeConfigurations = c.Env.ServiceDefinitionInput.VolumeConfigurations
	}
	if _, err := c.Ecs.UpdateService(ctx, updateInput); err != nil {
		return result, err
	}
	result.ServiceIntact = false
	log.Infof("waiting for service '%s' to be stable...", c.Env.Service)
	if err := ecs.NewServicesStableWaiter(c.Ecs).Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &c.Env.Cluster,
		Services: []string{c.Env.Service},
	}, c.Timeout.ServiceStable()); err != nil {
		return result, err
	}
	log.Infof("🥴 service '%s' has become to be stable!", c.Env.Service)
	log.Infof(
		"🐥 service '%s' successfully rolled out to '%s:%d'!",
		c.Env.Service, *nextTaskDefinition.Family, nextTaskDefinition.Revision,
	)
	return result, nil
}

func (c *cage) StartCanaryTasks(
	ctx context.Context,
	nextTaskDefinition *ecstypes.TaskDefinition,
	input *types.RollOutInput,
) ([]task.Task, error) {
	var networkConfiguration *ecstypes.NetworkConfiguration
	var platformVersion *string
	var loadBalancers []ecstypes.LoadBalancer
	var serviceRegistries []ecstypes.ServiceRegistry
	if input.UpdateService {
		networkConfiguration = c.Env.ServiceDefinitionInput.NetworkConfiguration
		platformVersion = c.Env.ServiceDefinitionInput.PlatformVersion
		loadBalancers = c.Env.ServiceDefinitionInput.LoadBalancers
		serviceRegistries = c.Env.ServiceDefinitionInput.ServiceRegistries
	} else {
		if o, err := c.Ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  &c.Env.Cluster,
			Services: []string{c.Env.Service},
		}); err != nil {
			return nil, err
		} else {
			service := o.Services[0]
			networkConfiguration = service.NetworkConfiguration
			platformVersion = service.PlatformVersion
			loadBalancers = service.LoadBalancers
			serviceRegistries = service.ServiceRegistries
		}
	}
	var results []task.Task
	for _, lb := range loadBalancers {
		task := c.TaskFactory.NewAlbTask(&task.Input{
			Deps:                 c.Deps,
			NetworkConfiguration: networkConfiguration,
			TaskDefinition:       nextTaskDefinition,
			PlatformVersion:      platformVersion,
			Timeout:              c.Timeout,
		}, &lb)
		results = append(results, task)
		if err := task.Start(ctx); err != nil {
			return results, err
		}
	}
	for _, srv := range serviceRegistries {
		task := c.TaskFactory.NewSrvTask(&task.Input{
			Deps:                 c.Deps,
			NetworkConfiguration: networkConfiguration,
			TaskDefinition:       nextTaskDefinition,
			PlatformVersion:      platformVersion,
			Timeout:              c.Timeout,
		}, &srv)
		results = append(results, task)
		if err := task.Start(ctx); err != nil {
			return results, err
		}
	}
	if len(results) == 0 {
		task := c.TaskFactory.NewSimpleTask(&task.Input{
			Deps:                 c.Deps,
			NetworkConfiguration: networkConfiguration,
			TaskDefinition:       nextTaskDefinition,
			PlatformVersion:      platformVersion,
			Timeout:              c.Timeout,
		})
		results = append(results, task)
		if err := task.Start(ctx); err != nil {
			return results, err
		}
	}
	return results, nil
}
