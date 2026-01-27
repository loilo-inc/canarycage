package task

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/mocks/mock_types"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewSimpleTask(t *testing.T) {
	d := di.NewDomain(func(b *di.B) {
		b.Set(key.Env, test.DefaultEnvars())
		b.Set(key.Logger, test.NewLogger())
	})
	task := NewSimpleTask(d, &Input{})
	v, ok := task.(*simpleTask)
	assert.NotNil(t, task)
	assert.True(t, ok)
	assert.Equal(t, d, v.di)
}

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
		b.Set(key.Logger, test.NewLogger())
		b.Set(key.Time, test.NewFakeTime())
	})
	stask := &simpleTask{
		common: &common{
			Input: &Input{
				TaskDefinition:       td.TaskDefinition,
				NetworkConfiguration: ecsSvc.Service.NetworkConfiguration,
			},
			di: d,
		},
	}
	err := stask.Start(ctx)
	assert.NoError(t, err)
	err = stask.Wait(ctx)
	assert.NoError(t, err)
	err = stask.Stop(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mocker.RunningTaskSize())
}

func TestSimpleTask_Wait(t *testing.T) {
	t.Run("should error if task is not running", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		cm := &simpleTask{
			common: &common{
				taskArn: aws.String("task-arn"),
				Input:   &Input{},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, test.DefaultEnvars())
					b.Set(key.EcsCli, ecsMock)
					b.Set(key.Logger, test.NewLogger())
				}),
			},
		}
		ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("STOPPED")}},
			}, nil)
		err := cm.Wait(context.TODO())
		assert.ErrorContains(t, err, "failed to wait for canary task to be running")
	})
	t.Run("should error if container is not healthy", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		mocker := test.NewMockContext()
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), test.DefaultEnvars().TaskDefinitionInput)
		env := test.DefaultEnvars()
		env.CanaryTaskHealthCheckWait = 1
		cm := &simpleTask{
			common: &common{
				taskArn: aws.String("task-arn"),
				Input:   &Input{TaskDefinition: td.TaskDefinition},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, env)
					b.Set(key.EcsCli, ecsMock)
					b.Set(key.Logger, test.NewLogger())
					b.Set(key.Time, test.NewFakeTime())
				}),
			},
		}
		ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("RUNNING"),
					Containers: []ecstypes.Container{{
						Name:         env.TaskDefinitionInput.ContainerDefinitions[0].Name,
						HealthStatus: ecstypes.HealthStatusUnhealthy,
					}},
				}},
			}, nil).Times(2)
		err := cm.Wait(context.TODO())
		assert.ErrorContains(t, err, "canary task hasn't become to be healthy")
	})
}

func TestSimpleTask_WaitForIdleDuration(t *testing.T) {
	setup := func(t *testing.T, idle int) (*mock_awsiface.MockEcsClient, *mock_types.MockTime, *simpleTask) {
		ctrl := gomock.NewController(t)
		mocker := test.NewMockContext()
		envars := test.DefaultEnvars()
		envars.CanaryTaskIdleDuration = idle
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), envars.TaskDefinitionInput)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		timerMock := mock_types.NewMockTime(ctrl)
		cm := &simpleTask{
			common: &common{
				Input: &Input{
					TaskDefinition: td.TaskDefinition,
				},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, envars)
					b.Set(key.EcsCli, ecsMock)
					b.Set(key.Logger, test.NewLogger())
					b.Set(key.Time, timerMock)
				}),
			},
		}
		cm.taskArn = aws.String("task-arn")
		return ecsMock, timerMock, cm
	}
	t.Run("should call DescribeTasks periodically", func(t *testing.T) {
		ecsMock, timerMock, cm := setup(t, 35)
		fakeTimer := test.NewFakeTime()
		gomock.InOrder(
			timerMock.EXPECT().NewTimer(15*time.Second).
				DoAndReturn(fakeTimer.NewTimer).
				Times(2),
			timerMock.EXPECT().NewTimer(5*time.Second).
				DoAndReturn(fakeTimer.NewTimer).
				Times(1),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).
				Return(&ecs.DescribeTasksOutput{
					Tasks: []ecstypes.Task{{LastStatus: aws.String("RUNNING")}},
				}, nil).
				Times(1),
		)
		err := cm.waitForIdleDuration(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should error if DescribeTasks failed", func(t *testing.T) {
		ecsMock, timerMock, cm := setup(t, 15)
		fakeTimer := test.NewFakeTime()
		gomock.InOrder(
			timerMock.EXPECT().NewTimer(15*time.Second).
				DoAndReturn(fakeTimer.NewTimer).
				Times(1),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).
				Return(nil, assert.AnError).
				Times(1),
		)
		err := cm.waitForIdleDuration(context.TODO())
		assert.EqualError(t, err, assert.AnError.Error())
	})
	t.Run("sholud error if ctx is canceled", func(t *testing.T) {
		_, timerMock, cm := setup(t, 15)
		gomock.InOrder(
			timerMock.EXPECT().NewTimer(15 * time.Second).
				DoAndReturn(time.NewTimer).
				Times(1),
		)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := cm.waitForIdleDuration(ctx)
		assert.EqualError(t, err, "context canceled")
	})
	t.Run("should error if task is not started", func(t *testing.T) {
		ecsMock, timerMock, cm := setup(t, 15)
		fakeTimer := test.NewFakeTime()
		gomock.InOrder(
			timerMock.EXPECT().NewTimer(15*time.Second).
				DoAndReturn(fakeTimer.NewTimer).
				Times(1),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).
				Return(&ecs.DescribeTasksOutput{
					Tasks: []ecstypes.Task{{
						LastStatus:    aws.String("STOPPED"),
						StoppedReason: aws.String("reason"),
					}},
				}, nil),
		)
		err := cm.waitForIdleDuration(context.TODO())
		assert.EqualError(t, err, "😫 canary task has stopped: reason")
	})
}
