package cage_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	cage "github.com/loilo-inc/canarycage"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCage_Run(t *testing.T) {
	setupForBasic := func(t *testing.T) (*env.Envars,
		*test.MockContext,
		*mock_awsiface.MockEcsClient) {
		env := test.DefaultEnvars()
		mocker := test.NewMockContext()
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.RegisterTaskDefinition).AnyTimes()
		return env, mocker, ecsMock
	}
	t.Run("basic", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, mocker, ecsMock := setupForBasic(t)
		gomock.InOrder(
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.RunTask),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.DescribeTasks),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, input *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
				mocker.Ecs.StopTask(ctx, &ecs.StopTaskInput{Cluster: &env.Cluster, Task: &input.Tasks[0]})
				return mocker.Ecs.DescribeTasks(ctx, input)
			}),
		)
		cagecli := cage.NewCage(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Time, test.NewFakeTime())
		}))
		result, err := cagecli.Run(ctx, &types.RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.NoError(t, err)
		assert.Equal(t, result.ExitCode, int32(0))
	})
	t.Run("should error if task failed to start", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, mocker, ecsMock := setupForBasic(t)
		env.CanaryTaskRunningWait = 1
		gomock.InOrder(
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.RunTask),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, input *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
					res, err := mocker.Ecs.DescribeTasks(ctx, input)
					for i := range res.Tasks {
						res.Tasks[i].LastStatus = aws.String("PROVISIONING")
					}
					return res, err
				},
			),
		)
		cagecli := cage.NewCage(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Time, test.NewFakeTime())
		}))
		result, err := cagecli.Run(ctx, &types.RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "task failed to start: exceeded max wait time for TasksRunning waiter")
	})
	t.Run("should error if task failed to stop", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, mocker, ecsMock := setupForBasic(t)
		env.CanaryTaskStoppedWait = 1
		gomock.InOrder(
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.RunTask),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.DescribeTasks).Times(2),
		)
		cagecli := cage.NewCage(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Time, test.NewFakeTime())
		}))
		result, err := cagecli.Run(ctx, &types.RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "task failed to stop: exceeded max wait time for TasksStopped waiter")
	})
	t.Run("should error if exit code was not 0", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, mocker, ecsMock := setupForBasic(t)
		gomock.InOrder(
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.RunTask),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.DescribeTasks),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, input *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
				stop, _ := mocker.Ecs.StopTask(ctx, &ecs.StopTaskInput{Cluster: &env.Cluster, Task: &input.Tasks[0]})
				for i := range stop.Task.Containers {
					stop.Task.Containers[i].ExitCode = aws.Int32(1)
				}
				return mocker.Ecs.DescribeTasks(ctx, input)
			}),
		)
		cagecli := cage.NewCage(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Time, test.NewFakeTime())
		}))
		result, err := cagecli.Run(ctx, &types.RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "task exited with 1")
	})
	t.Run("should error if exit code is nil", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		container := "container"
		ctx := context.Background()
		env, mocker, ecsMock := setupForBasic(t)
		gomock.InOrder(
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.RunTask),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.Ecs.DescribeTasks),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, input *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
				stop, _ := mocker.Ecs.StopTask(ctx, &ecs.StopTaskInput{Cluster: &env.Cluster, Task: &input.Tasks[0]})
				for i := range stop.Task.Containers {
					stop.Task.Containers[i].ExitCode = nil
				}
				return mocker.Ecs.DescribeTasks(ctx, input)
			}),
		)
		cagecli := cage.NewCage(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Time, test.NewFakeTime())
		}))
		result, err := cagecli.Run(ctx, &types.RunInput{
			Container: &container,
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "container 'container' hasn't exit")
	})
	t.Run("should error if container doesn't exist in definition", func(t *testing.T) {
		overrides := &ecstypes.TaskOverride{}
		ctx := context.Background()
		env, _, ecsMock := setupForBasic(t)
		cagecli := cage.NewCage(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Time, test.NewFakeTime())
		}))
		result, err := cagecli.Run(ctx, &types.RunInput{
			Container: aws.String("foo"),
			Overrides: overrides,
		})
		assert.Nil(t, result)
		assert.EqualError(t, err, "🚫 'foo' not found in container definitions")
	})
}
