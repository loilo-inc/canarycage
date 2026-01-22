package rollout

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/taskset"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
)

type Executor interface {
	RollOut(ctx context.Context, input *types.RollOutInput) error
	ServiceUpdated() bool
}

type executor struct {
	di             *di.D
	td             *ecstypes.TaskDefinition
	serviceUpdated bool
}

func NewExecutor(di *di.D, td *ecstypes.TaskDefinition) Executor {
	return &executor{di: di, td: td}
}

func (c *executor) RollOut(ctx context.Context, input *types.RollOutInput) (lastErr error) {
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	log := c.di.Get(key.Logger).(logger.Logger)
	if input.UpdateService {
		log.Infof("--updateService flag is set. use provided service configurations for canary test instead of current service")
	}
	canaryTasks, startCanaryTaskErr := c.startCanaryTasks(ctx, input)
	// ensure canary task stopped after rolling out either success or failure
	defer func() {
		_ = recover()
		if canaryTasks == nil {
			return
		} else if err := canaryTasks.Cleanup(ctx); err != nil {
			log.Errorf("failed to cleanup canary tasks due to: %s", err)
			lastErr = err
		}
	}()
	if startCanaryTaskErr != nil {
		log.Errorf("😨 failed to start canary task due to: %w", startCanaryTaskErr)
		return startCanaryTaskErr
	}
	log.Infof("executing canary tasks...")
	if err := canaryTasks.Exec(ctx); err != nil {
		log.Errorf("😨 failed to exec canary tasks: %s", err)
		return err
	}
	log.Infof("canary tasks have been executed successfully!")
	log.Infof(
		"updating the task definition of '%s' into '%s:%d'...",
		env.Service, *c.td.Family, c.td.Revision,
	)
	updateInput := &ecs.UpdateServiceInput{
		Cluster:        &env.Cluster,
		Service:        &env.Service,
		TaskDefinition: c.td.TaskDefinitionArn,
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
		log.Errorf("😨 failed to update service: %s", err)
		return err
	}
	c.serviceUpdated = true
	log.Infof("waiting for service '%s' to be stable...", env.Service)
	if err := ecs.NewServicesStableWaiter(ecsCli).Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  &env.Cluster,
		Services: []string{env.Service},
	}, env.GetServiceStableWait()); err != nil {
		log.Errorf("😨 failed to wait for service to be stable: %s", err)
		return err
	}
	log.Infof("🥴 service '%s' has become to be stable!", env.Service)
	log.Infof(
		"🐥 service '%s' successfully rolled out to '%s:%d'!",
		env.Service, *c.td.Family, c.td.Revision,
	)
	return nil
}

func (c *executor) ServiceUpdated() bool {
	return c.serviceUpdated
}

func (c *executor) startCanaryTasks(
	ctx context.Context,
	input *types.RollOutInput,
) (taskset.Set, error) {
	var networkConfiguration *ecstypes.NetworkConfiguration
	var platformVersion *string
	var loadBalancers []ecstypes.LoadBalancer
	env := c.di.Get(key.Env).(*env.Envars)
	factory := c.di.Get(key.TaskFactory).(task.Factory)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	if input.UpdateService {
		networkConfiguration = env.ServiceDefinitionInput.NetworkConfiguration
		platformVersion = env.ServiceDefinitionInput.PlatformVersion
		loadBalancers = env.ServiceDefinitionInput.LoadBalancers
	} else {
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
	return taskset.NewSet(
		factory,
		&taskset.Input{
			Input: &task.Input{
				NetworkConfiguration: networkConfiguration,
				TaskDefinition:       c.td,
				PlatformVersion:      platformVersion,
			},
			LoadBalancers: loadBalancers,
		},
	), nil
}
