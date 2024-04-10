package cage

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type RunInput struct {
	Container *string
	Overrides *types.TaskOverride
}
type RunResult struct {
	ExitCode int32
}

func containerExistsInDefinition(td *types.TaskDefinition, container *string) bool {
	for _, v := range td.ContainerDefinitions {
		if *v.Name == *container {
			return true
		}
	}
	return false
}

func (c *cage) Run(ctx context.Context, input *RunInput) (*RunResult, error) {
	td, err := c.CreateNextTaskDefinition(ctx)
	if err != nil {
		return nil, err
	}
	if !containerExistsInDefinition(td, input.Container) {
		return nil, fmt.Errorf("🚫 '%s' not found in container definitions", *input.Container)
	}
	o, err := c.ecs.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:              &c.env.Cluster,
		TaskDefinition:       td.TaskDefinitionArn,
		LaunchType:           types.LaunchTypeFargate,
		NetworkConfiguration: c.env.ServiceDefinitionInput.NetworkConfiguration,
		PlatformVersion:      c.env.ServiceDefinitionInput.PlatformVersion,
		Overrides:            input.Overrides,
	})
	if err != nil {
		return nil, err
	}
	taskArn := o.Tasks[0].TaskArn
	count := 0
	// 5min
	maxCount := 30
	interval := time.Second * 10
	var exitCode int32 = -1
	log.Infof("🤖 waiting until task '%s' is running...", *taskArn)
	for count < maxCount {
		<-newTimer(interval).C
		o, err := c.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &c.env.Cluster,
			Tasks:   []string{*taskArn},
		})
		if err != nil {
			return nil, err
		}
		task := o.Tasks[0]
		log.Infof("🤖 task status is '%s'", *task.LastStatus)
		if *task.LastStatus != "STOPPED" {
			count++
			continue
		}
		for _, container := range task.Containers {
			if *container.Name == *input.Container {
				if container.ExitCode == nil {
					return nil, fmt.Errorf("🚫 container '%s' hasn't exit", *input.Container)
				}
				exitCode = *container.ExitCode
				goto next
			}
		}
		return nil, fmt.Errorf("🚫 container '%s' not found in results", *input.Container)
	}
	return nil, fmt.Errorf("🚫 max attempts exceeded")
next:
	if exitCode != 0 {
		return nil, fmt.Errorf("🚫 task exited with %d", exitCode)
	}
	return &RunResult{ExitCode: exitCode}, nil
}
