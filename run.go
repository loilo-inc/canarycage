package cage

import (
	"context"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"golang.org/x/xerrors"
)

type RunInput struct {
	Container *string
	Overrides *types.TaskOverride
	MaxWait   time.Duration
}
type RunResult struct {
	ExitCode int32
}

func containerExistsInDefinition(td *ecs.RegisterTaskDefinitionInput, container *string) bool {
	for _, v := range td.ContainerDefinitions {
		if *v.Name == *container {
			return true
		}
	}
	return false
}

func (c *cage) Run(ctx context.Context, input *RunInput) (*RunResult, error) {
	if input.MaxWait == 0 {
		input.MaxWait = 5 * time.Minute
	}
	if !containerExistsInDefinition(c.Env.TaskDefinitionInput, input.Container) {
		return nil, xerrors.Errorf("🚫 '%s' not found in container definitions", *input.Container)
	}
	td, err := c.CreateNextTaskDefinition(ctx)
	if err != nil {
		return nil, err
	}
	o, err := c.Ecs.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:              &c.Env.Cluster,
		TaskDefinition:       td.TaskDefinitionArn,
		LaunchType:           types.LaunchTypeFargate,
		NetworkConfiguration: c.Env.ServiceDefinitionInput.NetworkConfiguration,
		PlatformVersion:      c.Env.ServiceDefinitionInput.PlatformVersion,
		Overrides:            input.Overrides,
		Group:                aws.String("cage:run-task"),
	})
	if err != nil {
		return nil, err
	}
	taskArn := o.Tasks[0].TaskArn
	log.Infof("waiting for task '%s' to start...", *taskArn)
	if err := ecs.NewTasksRunningWaiter(c.Ecs).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.Env.Cluster,
		Tasks:   []string{*taskArn},
	}, input.MaxWait); err != nil {
		return nil, xerrors.Errorf("task failed to start: %w", err)
	}
	log.Infof("task '%s' is running", *taskArn)
	log.Infof("waiting for task '%s' to stop...", *taskArn)
	if result, err := ecs.NewTasksStoppedWaiter(c.Ecs).WaitForOutput(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.Env.Cluster,
		Tasks:   []string{*taskArn},
	}, input.MaxWait); err != nil {
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
				return &RunResult{ExitCode: *c.ExitCode}, nil
			}
		}
		// Never reached?
		return nil, xerrors.Errorf("task '%s' not found in result", *taskArn)
	}
}
