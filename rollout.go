package cage

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/rollout"
	"github.com/loilo-inc/canarycage/types"
	"golang.org/x/xerrors"
)

func (c *cage) RollOut(ctx context.Context, input *types.RollOutInput) (*types.RollOutResult, error) {
	result := &types.RollOutResult{ServiceIntact: true}
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
	executor := rollout.NewExecutor(c.di, nextTaskDefinition)
	result.ServiceIntact = !executor.ServiceUpdated()
	err := executor.RollOut(ctx, input)
	return result, err
}
