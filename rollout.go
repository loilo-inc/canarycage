package cage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/v6/awsiface"
	"github.com/loilo-inc/canarycage/v6/env"
	"github.com/loilo-inc/canarycage/v6/key"
	"github.com/loilo-inc/canarycage/v6/logger"
	"github.com/loilo-inc/canarycage/v6/rollout"
	"github.com/loilo-inc/canarycage/v6/types"
)

func (c *cage) RollOut(ctx context.Context, input *types.RollOutInput) (*types.RollOutResult, error) {
	result, err := c.doRollOut(context.Background(), input)
	l := c.di.Get(key.Logger).(logger.Logger)
	e := c.di.Get(key.Env).(*env.Envars)
	if err != nil {
		if !result.ServiceUpdated {
			l.Errorf("🤕 failed to roll out new tasks but service '%s' is not changed", e.Service)
		} else {
			l.Errorf("😭 failed to roll out new tasks and service '%s' might be changed. CHECK ECS CONSOLE NOW!", e.Service)
		}
	} else {
		l.Infof("🎉 service roll out has completed successfully!🎉")
	}
	return result, err
}

func (c *cage) doRollOut(ctx context.Context, input *types.RollOutInput) (*types.RollOutResult, error) {
	result := &types.RollOutResult{}
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	if out, err := ecsCli.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &env.Cluster,
		Services: []string{env.Service},
	}); err != nil {
		return result, fmt.Errorf("failed to describe current service due to: %w", err)
	} else {
		var service *ecstypes.Service
		for _, s := range out.Services {
			if *s.ServiceName == env.Service {
				service = &s
				break
			}
		}
		if service == nil {
			return result, fmt.Errorf("service '%s' doesn't exist. Run 'cage up' or create service before rolling out", env.Service)
		}
		if *service.Status != "ACTIVE" {
			return result, fmt.Errorf("😵 service '%s' status is '%s'. Stop rolling out", env.Service, *service.Status)
		}
		if service.LaunchType == ecstypes.LaunchTypeEc2 && env.CanaryInstanceArn == "" {
			return result, fmt.Errorf("🥺 --canaryInstanceArn is required when LaunchType = 'EC2'")
		}
	}
	c.logger().Infof("ensuring next task definition...")
	var nextTaskDefinition *ecstypes.TaskDefinition
	if o, err := c.CreateNextTaskDefinition(ctx); err != nil {
		return result, fmt.Errorf("failed to register next task definition due to: %w", err)
	} else {
		nextTaskDefinition = o
	}
	executor := rollout.NewExecutor(c.di, nextTaskDefinition)
	err := executor.RollOut(ctx, input)
	result.ServiceUpdated = executor.ServiceUpdated()
	return result, err
}
