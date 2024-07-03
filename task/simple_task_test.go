package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/mocks/mock_types"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
)

func TestSimpleTask(t *testing.T) {
	ctx := context.TODO()
	mocker := test.NewMockContext()
	env := test.DefaultEnvars()
	td, _ := mocker.Ecs.RegisterTaskDefinition(ctx, env.TaskDefinitionInput)
	env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinition.TaskDefinitionArn
	env.CanaryTaskIdleDuration = 10
	ecsSvc, _ := mocker.Ecs.CreateService(ctx, env.ServiceDefinitionInput)
	d := di.NewDomain(func(b *di.B) {
		b.Set(key.Env, env)
		b.Set(key.EcsCli, mocker.Ecs)
		b.Set(key.Ec2Cli, mocker.Ec2)
		b.Set(key.AlbCli, mocker.Alb)
		b.Set(key.Time, test.NewFakeTime())
	})
	stask := task.NewSimpleTask(d, &task.Input{
		TaskDefinition:       td.TaskDefinition,
		NetworkConfiguration: ecsSvc.Service.NetworkConfiguration,
	})
	err := stask.Start(ctx)
	assert.NoError(t, err)
	err = stask.Wait(ctx)
	assert.NoError(t, err)
	err = stask.Stop(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mocker.RunningTaskSize())
}

func TestSimpleTask_WaitForIdleDuration(t *testing.T) {
	t.Run("should call DescribeTasks periodically", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mocker := test.NewMockContext()
		envars := test.DefaultEnvars()
		envars.CanaryTaskIdleDuration = 35 // sec
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), envars.TaskDefinitionInput)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		fakeTimer := test.NewFakeTime()
		timerMock := mock_types.NewMockTime(ctrl)
		gomock.InOrder(
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any()).
				DoAndReturn(mocker.Ecs.RunTask).
				Times(1),
			timerMock.EXPECT().NewTimer(15*time.Second).
				DoAndReturn(fakeTimer.NewTimer).
				Times(2),
			timerMock.EXPECT().NewTimer(5*time.Second).
				DoAndReturn(fakeTimer.NewTimer).
				Times(1),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).
				DoAndReturn(mocker.Ecs.DescribeTasks).
				Times(1),
		)
		cm := task.NewSimpleTaskExport(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, envars)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Time, timerMock)
		}), &task.Input{TaskDefinition: td.TaskDefinition})
		err := cm.Start(context.TODO())
		assert.NoError(t, err)
		err = cm.WaitForIdleDuration(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should error if DescribeTasks failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mocker := test.NewMockContext()
		envars := test.DefaultEnvars()
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), envars.TaskDefinitionInput)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		gomock.InOrder(
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any()).
				DoAndReturn(mocker.Ecs.RunTask).
				Times(1),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).
				Return(nil, assert.AnError).
				Times(1),
		)
		cm := task.NewSimpleTaskExport(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, envars)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Time, test.NewFakeTime())
		}), &task.Input{TaskDefinition: td.TaskDefinition})
		cm.Start(context.TODO())
		err := cm.WaitForIdleDuration(context.TODO())
		assert.EqualError(t, err, assert.AnError.Error())
	})
	t.Run("should error if task is not started", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mocker := test.NewMockContext()
		envars := test.DefaultEnvars()
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), envars.TaskDefinitionInput)
		ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any()).
			DoAndReturn(mocker.Ecs.RunTask).
			Times(1)
		ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).
			Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{
					LastStatus:    aws.String("STOPPED"),
					StoppedReason: aws.String("reason"),
				}},
			}, nil)
		cm := task.NewSimpleTaskExport(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, envars)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Time, test.NewFakeTime())
		}), &task.Input{TaskDefinition: td.TaskDefinition})
		cm.Start(context.TODO())
		err := cm.WaitForIdleDuration(context.TODO())
		assert.EqualError(t, err, "😫 canary task has stopped: reason")
	})
}
