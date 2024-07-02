package cage

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/taskset"
	"github.com/loilo-inc/canarycage/types"
	"golang.org/x/xerrors"
)

func (c *cage) RollOut(ctx context.Context, input *types.RollOutInput) (*types.RollOutResult, error) {
	result := &types.RollOutResult{
		ServiceIntact: true,
	}
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	if out, err := ecsCli.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &env.Cluster,
		Services: []string{env.Service},
	}); err != nil {
		return result, xerrors.Errorf("failed to describe current service due to: %w", err)
	} else if len(out.Services) == 0 {
		return result, xerrors.Errorf("service '%s' doesn't exist. Run 'cage up' or create service before rolling out", env.Service)
	} else {
		service := out.Services[0]
		if *service.Status != "ACTIVE" {
			return result, xerrors.Errorf("😵 '%s' status is '%s'. Stop rolling out", env.Service, *service.Status)
		}
		if service.LaunchType == ecstypes.LaunchTypeEc2 && env.CanaryInstanceArn == "" {
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
	canaryTasks, startCanaryTaskErr := c.StartCanaryTasks(ctx, nextTaskDefinition, input)
	// ensure canary task stopped after rolling out either success or failure
	defer func() {
		_ = recover()
		if canaryTasks == nil {
			return
		} else if err := canaryTasks.Cleanup(ctx); err != nil {
			log.Errorf("failed to cleanup canary tasks due to: %s", err)
		}
	}()
	if startCanaryTaskErr != nil {
		return result, xerrors.Errorf("failed to start canary task due to: %w", startCanaryTaskErr)
	}
	log.Infof("executing canary tasks...")
	if err := canaryTasks.Exec(ctx); err != nil {
		return result, xerrors.Errorf("failed to exec canary task due to: %w", err)
	}
	log.Infof("canary tasks have been executed successfully!")
	log.Infof(
		"updating the task definition of '%s' into '%s:%d'...",
		env.Service, *nextTaskDefinition.Family, nextTaskDefinition.Revision,
	)
	updateInput := &ecs.UpdateServiceInput{
		Cluster:        &env.Cluster,
		Service:        &env.Service,
		TaskDefinition: nextTaskDefinition.TaskDefinitionArn,
	}
	if input.UpdateService {
		updateInput.LoadBalancers = env.ServiceDefinitionInput.LoadBalancers
		updateInput.NetworkConfiguration = env.ServiceDefinitionInput.NetworkConfiguration
		updateInput.ServiceConnectConfiguration = env.ServiceDefinitionInput.ServiceConnectConfiguration
		updateInput.ServiceRegistries = env.ServiceDefinitionInput.ServiceRegistries
		updateInput.PlatformVersion = env.ServiceDefinitionInput.PlatformVersion
		updateInput.VolumeConfigurations = env.ServiceDefinitionInput.VolumeConfigurations
	}
	if _, err := ecsCli.UpdateService(ctx, updateInput); err != nil {
		return result, err
	}
	result.ServiceIntact = false
	log.Infof("waiting for service '%s' to be stable...", env.Service)
	if err := ecs.NewServicesStableWaiter(ecsCli).Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &env.Cluster,
		Services: []string{env.Service},
	}, env.GetServiceStableWait()); err != nil {
		return result, err
	}
	log.Infof("🥴 service '%s' has become to be stable!", env.Service)
	log.Infof(
		"🐥 service '%s' successfully rolled out to '%s:%d'!",
		env.Service, *nextTaskDefinition.Family, nextTaskDefinition.Revision,
	)
	return result, nil
}

func (c *cage) StartCanaryTasks(
	ctx context.Context,
	nextTaskDefinition *ecstypes.TaskDefinition,
	input *types.RollOutInput,
) (taskset.Set, error) {
	var networkConfiguration *ecstypes.NetworkConfiguration
	var platformVersion *string
	var loadBalancers []ecstypes.LoadBalancer
	env := c.di.Get(key.Env).(*env.Envars)
	if input.UpdateService {
		networkConfiguration = env.ServiceDefinitionInput.NetworkConfiguration
		platformVersion = env.ServiceDefinitionInput.PlatformVersion
		loadBalancers = env.ServiceDefinitionInput.LoadBalancers
	} else {
		ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
		if o, err := ecsCli.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  &env.Cluster,
			Services: []string{env.Service},
		}); err != nil {
			return nil, err
		} else {
			service := o.Services[0]
			networkConfiguration = service.NetworkConfiguration
			platformVersion = service.PlatformVersion
			loadBalancers = service.LoadBalancers
		}
	}
	factory := c.di.Get(key.TaskFactory).(task.Factory)
	return taskset.NewSet(
		factory,
		&taskset.Input{
			Input: &task.Input{
				NetworkConfiguration: networkConfiguration,
				TaskDefinition:       nextTaskDefinition,
				PlatformVersion:      platformVersion,
			},
			LoadBalancers: loadBalancers,
		},
	), nil
}
