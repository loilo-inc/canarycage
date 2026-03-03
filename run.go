package cage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/v5/awsiface"
	"github.com/loilo-inc/canarycage/v5/env"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/types"
)

func containerExistsInDefinition(td *ecs.RegisterTaskDefinitionInput, container string) bool {
	for _, v := range td.ContainerDefinitions {
		if *v.Name == container {
			return true
		}
	}
	return false
}

func (c *cage) Run(ctx context.Context, input *types.RunInput) (*types.RunResult, error) {
	result, err := c.doRun(ctx, input)
	l := c.logger()
	if err != nil {
		l.Errorf("🤕 task execution failed: %v", err)
		return nil, err
	}
	l.Infof("👍 task successfully executed")
	return result, nil
}

func (c *cage) doRun(ctx context.Context, input *types.RunInput) (*types.RunResult, error) {
	env := c.di.Get(key.Env).(*env.Envars)
	if !containerExistsInDefinition(env.TaskDefinitionInput, input.Container) {
		return nil, fmt.Errorf("🚫 '%s' not found in container definitions", input.Container)
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

	// NOTE: https://github.com/loilo-inc/canarycage/issues/93
	// wait for the task to be running
	t := c.di.Get(key.Time).(types.Time)
	<-t.NewTimer(2 * time.Second).C

	l := c.logger()
	l.Infof("waiting for task '%s' to start...", *taskArn)
	if waitErr := ecs.NewTasksRunningWaiter(ecsCli).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*taskArn},
	}, env.GetTaskRunningWait()); waitErr != nil {
		l.Infof("task '%s' might have failed to start. try to check container status: %v", *taskArn, waitErr)
		desc, descErr := ecsCli.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &env.Cluster,
			Tasks:   []string{*taskArn},
		})
		if descErr != nil {
			return nil, fmt.Errorf("failed to describe task: %w", descErr)
		} else if len(desc.Tasks) == 0 {
			return nil, fmt.Errorf("task failed to start: no tasks found")
		}
		task := desc.Tasks[0]
		if task.LastStatus != nil && *task.LastStatus != "STOPPED" {
			return nil, fmt.Errorf("task failed to start: task is in '%s' status", *task.LastStatus)
		} else if res, err := checkTaskStopped(task, input.Container); err != nil {
			return nil, fmt.Errorf("task failed to start: %w", err)
		} else {
			return res, nil
		}
	}
	l.Infof("task '%s' is running", *taskArn)
	l.Infof("waiting for task '%s' to stop...", *taskArn)
	if result, err := ecs.NewTasksStoppedWaiter(ecsCli).WaitForOutput(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*taskArn},
	}, env.GetTaskStoppedWait()); err != nil {
		return nil, fmt.Errorf("task failed to stop: %w", err)
	} else {
		return checkTaskStopped(result.Tasks[0], input.Container)
	}
}

func checkTaskStopped(task ecstypes.Task, runContainer string) (*types.RunResult, error) {
	for _, c := range task.Containers {
		if *c.Name != runContainer {
			continue
		}
		if c.ExitCode == nil {
			return nil, fmt.Errorf("container '%s' hasn't exited", *c.Name)
		}
		exitCode := *c.ExitCode
		if exitCode != 0 {
			return nil, fmt.Errorf("task exited with %d", exitCode)
		} else {
			return &types.RunResult{ExitCode: exitCode}, nil
		}
	}
	return nil, fmt.Errorf("container '%s' not found in task", runContainer)
}
