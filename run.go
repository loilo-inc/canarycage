package cage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/canarycage/types"
	"golang.org/x/xerrors"
)

func containerExistsInDefinition(td *ecs.RegisterTaskDefinitionInput, container *string) bool {
	for _, v := range td.ContainerDefinitions {
		if *v.Name == *container {
			return true
		}
	}
	return false
}

func (c *cage) Run(ctx context.Context, input *types.RunInput) (*types.RunResult, error) {
	result, err := c.doRun(ctx, input)
	log := c.di.Get(key.Logger).(logger.Logger)
	if err != nil {
		return nil, err
	}
	log.Infof("👍 task successfully executed")
	return result, nil
}

func (c *cage) doRun(ctx context.Context, input *types.RunInput) (*types.RunResult, error) {
	env := c.di.Get(key.Env).(*env.Envars)
	if !containerExistsInDefinition(env.TaskDefinitionInput, input.Container) {
		return nil, xerrors.Errorf("🚫 '%s' not found in container definitions", *input.Container)
	}
	td, err := c.CreateNextTaskDefinition(ctx)
	if err != nil {
		return nil, err
	}
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	o, err := ecsCli.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:              &env.Cluster,
		TaskDefinition:       td.TaskDefinitionArn,
		LaunchType:           ecstypes.LaunchTypeFargate,
		NetworkConfiguration: env.ServiceDefinitionInput.NetworkConfiguration,
		PlatformVersion:      env.ServiceDefinitionInput.PlatformVersion,
		Overrides:            input.Overrides,
		Group:                aws.String("cage:run-task"),
	})
	if err != nil {
		return nil, err
	}
	taskArn := o.Tasks[0].TaskArn
	log := c.di.Get(key.Logger).(logger.Logger)
	// NOTE: https://github.com/loilo-inc/canarycage/issues/93
	// wait for the task to be running
	time.Sleep(2 * time.Second)

	log.Infof("waiting for task '%s' to start...", *taskArn)
	if err := ecs.NewTasksRunningWaiter(ecsCli).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*taskArn},
	}, env.GetTaskRunningWait()); err != nil {
		return nil, xerrors.Errorf("task failed to start: %w", err)
	}
	log.Infof("task '%s' is running", *taskArn)
	log.Infof("waiting for task '%s' to stop...", *taskArn)
	if result, err := ecs.NewTasksStoppedWaiter(ecsCli).WaitForOutput(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*taskArn},
	}, env.GetTaskStoppedWait()); err != nil {
		return nil, xerrors.Errorf("task failed to stop: %w", err)
	} else {
		task := result.Tasks[0]
		for _, c := range task.Containers {
			if *c.Name == *input.Container {
				if c.ExitCode == nil {
					return nil, xerrors.Errorf("container '%s' hasn't exit", *input.Container)
				} else if *c.ExitCode != 0 {
					return nil, xerrors.Errorf("task exited with %d", *c.ExitCode)
				}
				return &types.RunResult{ExitCode: *c.ExitCode}, nil
			}
		}
		// Never reached?
		return nil, xerrors.Errorf("task '%s' not found in result", *taskArn)
	}
}
