package cage_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/golang/mock/gomock"
	cage "github.com/loilo-inc/canarycage"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/test"
	"github.com/stretchr/testify/assert"
)

func TestCage_Run(t *testing.T) {
	setupForBasic := func(t *testing.T) (*cage.Envars, *mock_awsiface.MockEcsClient) {
		env := test.DefaultEnvars()
		mocker := test.NewMockContext()
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTaskDefinition).AnyTimes()
		ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeTasks).AnyTimes()
		return env, ecsMock
	}
	t.Run("basic", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, ecsMock := setupForBasic(t)
		ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, input *ecs.RunTaskInput, optFns ...func(*ecs.Options)) (*ecs.RunTaskOutput, error) {
				task, err := mocker.RunTask(ctx, input)
				if err != nil {
					return nil, err
				}
				stop, err := mocker.StopTask(ctx, &ecs.StopTaskInput{Cluster: input.Cluster, Task: task.Tasks[0].TaskArn})
				if err != nil {
					return nil, err
				}
				return &ecs.RunTaskOutput{Tasks: []ecstypes.Task{*stop.Task}}, nil
			},
		).AnyTimes()
		cagecli := cage.NewCage(&cage.Input{
			Env:  env,
			ECS:  ecsMock,
			ALB:  nil,
			EC2:  nil,
			Time: test.NewFakeTime(),
		})
		result, err := cagecli.Run(ctx, &cage.RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.NoError(t, err)
		assert.Equal(t, result.ExitCode, int32(0))
	})
	t.Run("should error if max attempts exceeded", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, ecsMock := setupForBasic(t)
		ecsMock.EXPECT().DescribeTasks(ctx, gomock.Any()).AnyTimes().Return(&ecs.DescribeTasksOutput{
			Tasks: []ecstypes.Task{
				{LastStatus: aws.String("RUNNING"),
					Containers: []ecstypes.Container{{
						Name:     &container,
						ExitCode: nil,
					}}},
			},
		}, nil)
		cagecli := cage.NewCage(&cage.Input{
			Env:  env,
			ECS:  ecsMock,
			ALB:  nil,
			EC2:  nil,
			Time: test.NewFakeTime(),
		})
		result, err := cagecli.Run(ctx, &cage.RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "🚫 max attempts exceeded")
	})
	t.Run("should error if exit code was not 0", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, ecsMock := setupForBasic(t)
		cagecli := cage.NewCage(&cage.Input{
			Env:  env,
			ECS:  ecsMock,
			ALB:  nil,
			EC2:  nil,
			Time: test.NewFakeTime(),
		})
		result, err := cagecli.Run(ctx, &cage.RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "🚫 task exited with 1")
	})
	t.Run("should error if exit code is nil", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, ecsMock := setupForBasic(t)
		cagecli := cage.NewCage(&cage.Input{
			Env:  env,
			ECS:  ecsMock,
			ALB:  nil,
			EC2:  nil,
			Time: test.NewFakeTime(),
		})
		result, err := cagecli.Run(ctx, &cage.RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "🚫 container 'container' hasn't exit")
	})
	t.Run("should error if container doesn't exist in definition", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, ecsMock := setupForBasic(t)
		td := &ecs.RegisterTaskDefinitionOutput{
			TaskDefinition: &ecstypes.TaskDefinition{
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{Name: &container},
				},
			},
		}

		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).Return(td, nil)
		cagecli := cage.NewCage(&cage.Input{
			Env:  env,
			ECS:  ecsMock,
			ALB:  nil,
			EC2:  nil,
			Time: test.NewFakeTime(),
		})
		result, err := cagecli.Run(ctx, &cage.RunInput{
			Container: aws.String("foo"),
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "🚫 'foo' not found in container definitions")
	})
}
