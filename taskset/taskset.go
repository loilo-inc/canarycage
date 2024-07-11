package taskset

import (
	"context"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/task"
	"golang.org/x/sync/errgroup"
)

type Set interface {
	Exec(ctx context.Context) error
	Cleanup(ctx context.Context) error
}

type set struct {
	tasks []task.Task
}

type Input struct {
	*task.Input
	LoadBalancers []ecstypes.LoadBalancer
}

func NewSet(
	factory task.Factory,
	input *Input) Set {
	var results []task.Task
	taskInput := &task.Input{
		NetworkConfiguration: input.NetworkConfiguration,
		TaskDefinition:       input.TaskDefinition,
		PlatformVersion:      input.PlatformVersion,
	}
	for _, lb := range input.LoadBalancers {
		task := factory.NewAlbTask(taskInput, &lb)
		results = append(results, task)
	}
	if len(results) == 0 {
		task := factory.NewSimpleTask(taskInput)
		results = append(results, task)
	}
	return &set{tasks: results}
}

func (s *set) Exec(ctx context.Context) error {
	for _, t := range s.tasks {
		if err := t.Start(ctx); err != nil {
			return err
		}
	}
	eg := errgroup.Group{}
	for _, t := range s.tasks {
		eg.Go(func() error {
			return t.Wait(ctx)
		})
	}
	return eg.Wait()
}

func (s *set) Cleanup(ctx context.Context) error {
	eg := errgroup.Group{}
	for _, t := range s.tasks {
		eg.Go(func() error {
			return t.Stop(ctx)
		})
	}
	return eg.Wait()
}
