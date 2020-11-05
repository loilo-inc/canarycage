package cage

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/golang/mock/gomock"
	mock_ecsiface "github.com/loilo-inc/canarycage/mocks/github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCage_Run(t *testing.T) {
	container := "container"
	setupForBasic := func(ctx context.Context,
		ctrl *gomock.Controller,
		results []*ecs.DescribeTasksOutput) *mock_ecsiface.MockECSAPI {
		ecsMock := mock_ecsiface.NewMockECSAPI(ctrl)
		td := &ecs.RegisterTaskDefinitionOutput{
			TaskDefinition: &ecs.TaskDefinition{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{Name: &container},
				},
			},
		}
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any()).Return(td, nil)
		runTaskOutput := &ecs.RunTaskOutput{
			Tasks: []*ecs.Task{
				{TaskArn: aws.String("arn")},
			},
		}
		ecsMock.EXPECT().RunTaskWithContext(ctx, gomock.Any()).Return(runTaskOutput, nil)
		for _, o := range results {
			ecsMock.EXPECT().DescribeTasksWithContext(ctx, gomock.Any()).Return(o, nil)
		}
		return ecsMock
	}
	t.Run("basic", func(t *testing.T) {
		env := DefaultEnvars()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		overrides := &ecs.TaskOverride{}
		container := "container"
		ctx := context.Background()
		ecsMock := setupForBasic(ctx, ctrl, []*ecs.DescribeTasksOutput{
			{Tasks: []*ecs.Task{
				{LastStatus: aws.String("RUNNING"),
					Containers: []*ecs.Container{{
						Name:     &container,
						ExitCode: nil,
					}}},
			}},
			{Tasks: []*ecs.Task{
				{LastStatus: aws.String("STOPPED"),
					Containers: []*ecs.Container{{
						Name:     &container,
						ExitCode: aws.Int64(0),
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
		assert.Equal(t, result.ExitCode, int64(0))
	})
	t.Run("should error if max attempts exceeded", func(t *testing.T) {
		env := DefaultEnvars()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		overrides := &ecs.TaskOverride{}
		container := "container"
		ctx := context.Background()
		ecsMock := setupForBasic(ctx, ctrl, nil)
		ecsMock.EXPECT().DescribeTasksWithContext(ctx, gomock.Any()).AnyTimes().Return(&ecs.DescribeTasksOutput{
			Tasks: []*ecs.Task{
				{LastStatus: aws.String("RUNNING"),
					Containers: []*ecs.Container{{
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
		overrides := &ecs.TaskOverride{}
		container := "container"
		ctx := context.Background()
		ecsMock := setupForBasic(ctx, ctrl, []*ecs.DescribeTasksOutput{
			{Tasks: []*ecs.Task{
				{LastStatus: aws.String("STOPPED"),
					Containers: []*ecs.Container{{
						Name:     &container,
						ExitCode: aws.Int64(1),
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
	t.Run("should error if container doesn't exist in definition", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		env := DefaultEnvars()
		overrides := &ecs.TaskOverride{}
		container := "container"
		ctx := context.Background()
		ecsMock := mock_ecsiface.NewMockECSAPI(ctrl)
		td := &ecs.RegisterTaskDefinitionOutput{
			TaskDefinition: &ecs.TaskDefinition{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{Name: &container},
				},
			},
		}

		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any()).Return(td, nil)
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
		assert.EqualError(t, err, "'foo' not found in container definitions")
	})
}
