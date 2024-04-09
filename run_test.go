package cage

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/stretchr/testify/assert"
)

func TestCage_Run(t *testing.T) {
	container := "container"
	setupForBasic := func(ctx context.Context,
		ctrl *gomock.Controller,
		results []*ecs.DescribeTasksOutput) *mock_awsiface.MockEcsClient {
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		td := &ecs.RegisterTaskDefinitionOutput{
			TaskDefinition: &ecstypes.TaskDefinition{
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{Name: &container},
				},
			},
		}
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).Return(td, nil)
		runTaskOutput := &ecs.RunTaskOutput{
			Tasks: []ecstypes.Task{
				{TaskArn: aws.String("arn")},
			},
		}
		ecsMock.EXPECT().RunTask(ctx, gomock.Any()).Return(runTaskOutput, nil)
		for _, o := range results {
			ecsMock.EXPECT().DescribeTasks(ctx, gomock.Any()).Return(o, nil)
		}
		return ecsMock
	}
	t.Run("basic", func(t *testing.T) {
		env := DefaultEnvars()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		ecsMock := setupForBasic(ctx, ctrl, []*ecs.DescribeTasksOutput{
			{Tasks: []ecstypes.Task{
				{LastStatus: aws.String("RUNNING"),
					Containers: []ecstypes.Container{{
						Name:     &container,
						ExitCode: nil,
					}}},
			}},
			{Tasks: []ecstypes.Task{
				{LastStatus: aws.String("STOPPED"),
					Containers: []ecstypes.Container{{
						Name:     &container,
						ExitCode: aws.Int32(0),
					}},
				},
			}},
		})

		cagecli := NewCage(&Input{
			Env: env,
			ECS: ecsMock,
			ALB: nil,
			EC2: nil,
		})
		newTimer = fakeTimer
		defer recoverTimer()
		result, err := cagecli.Run(ctx, &RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, err)
		assert.Equal(t, result.ExitCode, int32(0))
	})
	t.Run("should error if max attempts exceeded", func(t *testing.T) {
		env := DefaultEnvars()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		ecsMock := setupForBasic(ctx, ctrl, nil)
		ecsMock.EXPECT().DescribeTasks(ctx, gomock.Any()).AnyTimes().Return(&ecs.DescribeTasksOutput{
			Tasks: []ecstypes.Task{
				{LastStatus: aws.String("RUNNING"),
					Containers: []ecstypes.Container{{
						Name:     &container,
						ExitCode: nil,
					}}},
			},
		}, nil)
		cagecli := NewCage(&Input{
			Env: env,
			ECS: ecsMock,
			ALB: nil,
			EC2: nil,
		})
		newTimer = fakeTimer
		defer recoverTimer()
		result, err := cagecli.Run(ctx, &RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "🚫 max attempts exceeded")
	})
	t.Run("should error if exit code was not 0", func(t *testing.T) {
		env := DefaultEnvars()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		ecsMock := setupForBasic(ctx, ctrl, []*ecs.DescribeTasksOutput{
			{Tasks: []ecstypes.Task{
				{LastStatus: aws.String("STOPPED"),
					Containers: []ecstypes.Container{{
						Name:     &container,
						ExitCode: aws.Int32(1),
					}}},
			}},
		})
		cagecli := NewCage(&Input{
			Env: env,
			ECS: ecsMock,
			ALB: nil,
			EC2: nil,
		})
		newTimer = fakeTimer
		defer recoverTimer()
		result, err := cagecli.Run(ctx, &RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "🚫 task exited with 1")
	})
	t.Run("should error if exit code is nil", func(t *testing.T) {
		env := DefaultEnvars()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		ecsMock := setupForBasic(ctx, ctrl, []*ecs.DescribeTasksOutput{
			{Tasks: []ecstypes.Task{
				{LastStatus: aws.String("STOPPED"),
					Containers: []ecstypes.Container{{
						Name:     &container,
						ExitCode: nil,
					}}},
			}},
		})
		cagecli := NewCage(&Input{
			Env: env,
			ECS: ecsMock,
			ALB: nil,
			EC2: nil,
		})
		newTimer = fakeTimer
		defer recoverTimer()
		result, err := cagecli.Run(ctx, &RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "🚫 container 'container' hasn't exit")
	})
	t.Run("should error if container doesn't exist in definition", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		env := DefaultEnvars()
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		td := &ecs.RegisterTaskDefinitionOutput{
			TaskDefinition: &ecstypes.TaskDefinition{
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{Name: &container},
				},
			},
		}

		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).Return(td, nil)
		cagecli := NewCage(&Input{
			Env: env,
			ECS: ecsMock,
			ALB: nil,
			EC2: nil,
		})
		result, err := cagecli.Run(ctx, &RunInput{
			Container: aws.String("foo"),
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "🚫 'foo' not found in container definitions")
	})
}
