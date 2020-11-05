package cage

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"time"
)

type RunInput struct {
	Container *string
	Overrides *ecs.TaskOverride
}
type RunResult struct {
	ExitCode int64
}

func (c *cage) Run(ctx context.Context, input *RunInput) (*RunResult, error) {
	td, err := c.CreateNextTaskDefinition()
	if err != nil {
		return nil, err
	}
	o, err := c.ecs.RunTaskWithContext(ctx, &ecs.RunTaskInput{
		Cluster:              &c.env.Cluster,
		TaskDefinition:       td.TaskDefinitionArn,
		LaunchType:           aws.String("FARGATE"),
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
	var exitCode int64 = -1
	log.Infof("🥚 waiting until task '%s' is running...", *taskArn)
	for count < maxCount {
		<-time.NewTimer(interval).C
		o, err := c.ecs.DescribeTasks(&ecs.DescribeTasksInput{
			Cluster: &c.env.Cluster,
			Tasks:   []*string{taskArn},
		})
		if err != nil {
			return nil, err
		}
		task := o.Tasks[0]
		if *task.LastStatus != "STOPPED" {
			count++
			continue
		}
		for _, container := range task.Containers {
			if *container.Name == *input.Container {
				exitCode = *container.ExitCode
				goto next
			}
			return nil, fmt.Errorf("container \"%s\" not found in results", *input.Container)
		}
	}
next:
	if exitCode != 0 {
		return nil, fmt.Errorf("🚫 task exited with %d", exitCode)
	}
	return &RunResult{ExitCode: exitCode}, nil
}
