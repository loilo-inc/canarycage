package task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/mocks/mock_types"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCommon_Start(t *testing.T) {
	t.Run("Fargate", func(t *testing.T) {
		t.Run("basic", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			td := &ecstypes.TaskDefinition{}
			ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ecs.RunTaskOutput{
				Tasks: []ecstypes.Task{{TaskArn: aws.String("task-arn")}},
			}, nil)
			envars := test.DefaultEnvars()
			cm := &common{
				Input: &Input{TaskDefinition: td},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, envars)
					b.Set(key.EcsCli, ecsMock)
					b.Set(key.Logger, test.NewLogger())
				}),
			}
			err := cm.Start(context.TODO())
			assert.NoError(t, err)
		})
		t.Run("should error if task failed to start", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			td := &ecstypes.TaskDefinition{}
			ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
			envars := test.DefaultEnvars()
			cm := &common{
				Input: &Input{TaskDefinition: td},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, envars)
					b.Set(key.EcsCli, ecsMock)
					b.Set(key.Logger, test.NewLogger())
				}),
			}
			err := cm.Start(context.TODO())
			assert.EqualError(t, err, "error")
		})
	})
	t.Run("EC2", func(t *testing.T) {
		t.Run("basic", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			td := &ecstypes.TaskDefinition{}
			ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
			ecsMock.EXPECT().StartTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ecs.StartTaskOutput{
				Tasks: []ecstypes.Task{{TaskArn: aws.String("task-arn")}},
			}, nil)
			envars := test.DefaultEnvars()
			envars.CanaryInstanceArn = "instance-arn"
			cm := &common{
				Input: &Input{TaskDefinition: td},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, envars)
					b.Set(key.EcsCli, ecsMock)
					b.Set(key.Logger, test.NewLogger())
				}),
			}
			err := cm.Start(context.TODO())
			assert.NoError(t, err)
		})
		t.Run("should error if task failed to start", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			td := &ecstypes.TaskDefinition{}
			ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
			ecsMock.EXPECT().StartTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
			envars := test.DefaultEnvars()
			envars.CanaryInstanceArn = "instance-arn"
			cm := &common{
				Input: &Input{TaskDefinition: td},
				di: di.NewDomain(func(b *di.B) {
					b.Set(key.Env, envars)
					b.Set(key.EcsCli, ecsMock)
					b.Set(key.Logger, test.NewLogger())
				}),
			}
			err := cm.Start(context.TODO())
			assert.EqualError(t, err, "error")
		})
	})
}

func TestCommon_WaitForTaskRunning(t *testing.T) {
	setup := func(t *testing.T, envars *env.Envars) (*mock_awsiface.MockEcsClient, *common) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		td := &ecstypes.TaskDefinition{}
		cm := &common{
			Input: &Input{TaskDefinition: td},
			di: di.NewDomain(func(b *di.B) {
				b.Set(key.Env, envars)
				b.Set(key.EcsCli, ecsMock)
				b.Set(key.Logger, test.NewLogger())
			}),
		}
		cm.taskArn = aws.String("task-arn")
		return ecsMock, cm
	}
	t.Run("should call ecs.NewTasksRunningWaiter", func(t *testing.T) {
		ecsMock, cm := setup(t, test.DefaultEnvars())
		ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
			Tasks: []ecstypes.Task{{LastStatus: aws.String("RUNNING")}},
		}, nil)
		err := cm.waitForTaskRunning(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should error if task is not started", func(t *testing.T) {
		cm := &common{}
		err := cm.waitForTaskRunning(context.TODO())
		assert.EqualError(t, err, "task is not started")
	})
	t.Run("should error if ecs.NewTasksRunningWaiter failed", func(t *testing.T) {
		envars := test.DefaultEnvars()
		envars.CanaryTaskRunningWait = 15
		ecsMock, cm := setup(t, envars)
		ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("STOPPED"), StoppedReason: aws.String("reason")}},
			}, nil)
		err := cm.waitForTaskRunning(context.TODO())
		assert.ErrorContains(t, err, "failed to wait for canary task to be running:")
	})
}

func TestCommon_WaitContainerHealthCheck(t *testing.T) {
	setup := func(t *testing.T, envars *env.Envars) (*mock_awsiface.MockEcsClient, *mock_types.MockTime,
		*ecstypes.TaskDefinition,
		*common) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		mocker := test.NewMockContext()
		timerMock := mock_types.NewMockTime(ctrl)
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), envars.TaskDefinitionInput)
		cm := &common{
			Input: &Input{TaskDefinition: td.TaskDefinition},
			di: di.NewDomain(func(b *di.B) {
				b.Set(key.Env, envars)
				b.Set(key.EcsCli, ecsMock)
				b.Set(key.Logger, test.NewLogger())
				b.Set(key.Time, timerMock)
			}),
		}
		cm.taskArn = aws.String("task-arn")
		return ecsMock, timerMock, td.TaskDefinition, cm
	}
	t.Run("should call DescribeTasks periodically", func(t *testing.T) {
		env := test.DefaultEnvars()
		env.CanaryTaskHealthCheckWait = 20
		ecsMock, timerMock, td, cm := setup(t, env)
		faketime := test.NewFakeTime()
		gomock.InOrder(
			timerMock.EXPECT().NewTimer(15*time.Second).DoAndReturn(faketime.NewTimer),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("RUNNING"),
					Containers: []ecstypes.Container{
						{Name: td.ContainerDefinitions[0].Name,
							HealthStatus: ecstypes.HealthStatusUnknown},
						{Name: td.ContainerDefinitions[1].Name,
							HealthStatus: ecstypes.HealthStatusUnknown},
					},
				}},
			}, nil),
			timerMock.EXPECT().NewTimer(5*time.Second).DoAndReturn(faketime.NewTimer),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("RUNNING"),
					Containers: []ecstypes.Container{
						{Name: td.ContainerDefinitions[0].Name,
							HealthStatus: ecstypes.HealthStatusHealthy},
						{Name: td.ContainerDefinitions[1].Name,
							HealthStatus: ecstypes.HealthStatusUnknown},
					},
				}},
			}, nil),
		)
		err := cm.waitContainerHealthCheck(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should do nothing if no container has health check", func(t *testing.T) {
		env := test.DefaultEnvars()
		env.TaskDefinitionInput.ContainerDefinitions[0].HealthCheck = nil
		_, _, _, cm := setup(t, env)
		err := cm.waitContainerHealthCheck(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should error if DescribeTasks failed", func(t *testing.T) {
		env := test.DefaultEnvars()
		ecsMock, timerMock, _, cm := setup(t, env)
		faketime := test.NewFakeTime()
		gomock.InOrder(
			timerMock.EXPECT().NewTimer(15*time.Second).DoAndReturn(faketime.NewTimer),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")),
		)
		err := cm.waitContainerHealthCheck(context.TODO())
		assert.EqualError(t, err, "error")
	})
	t.Run("should error if context is canceled", func(t *testing.T) {
		env := test.DefaultEnvars()
		_, timerMock, _, cm := setup(t, env)
		timerMock.EXPECT().NewTimer(15 * time.Second).DoAndReturn(time.NewTimer)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := cm.waitContainerHealthCheck(ctx)
		assert.EqualError(t, err, "context canceled")
	})
	t.Run("should error if task is not running", func(t *testing.T) {
		env := test.DefaultEnvars()
		ecsMock, timerMock, _, cm := setup(t, env)
		faketime := test.NewFakeTime()
		gomock.InOrder(
			timerMock.EXPECT().NewTimer(15*time.Second).DoAndReturn(faketime.NewTimer),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("STOPPED"),
					StoppedReason: aws.String("reason")},
				},
			},
				nil),
		)
		err := cm.waitContainerHealthCheck(context.TODO())
		assert.EqualError(t, err, "😫 canary task has stopped: reason")
	})
	t.Run("shold error if unhealth counts exceed the limit", func(t *testing.T) {
		env := test.DefaultEnvars()
		env.CanaryTaskHealthCheckWait = 15
		ecsMock, timerMock, td, cm := setup(t, env)
		faketime := test.NewFakeTime()
		gomock.InOrder(
			timerMock.EXPECT().NewTimer(15*time.Second).DoAndReturn(faketime.NewTimer),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("RUNNING"),
					Containers: []ecstypes.Container{
						{Name: td.ContainerDefinitions[0].Name,
							HealthStatus: ecstypes.HealthStatusUnhealthy},
						{Name: td.ContainerDefinitions[1].Name,
							HealthStatus: ecstypes.HealthStatusUnknown},
					},
				}},
			}, nil),
		)
		err := cm.waitContainerHealthCheck(context.TODO())
		assert.EqualError(t, err, "😨 canary task hasn't become to be healthy")
	})
}

func TestCommon_StopTask(t *testing.T) {
	setup := func(t *testing.T, env *env.Envars) (*mock_awsiface.MockEcsClient, *common) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		cm := &common{
			di: di.NewDomain(func(b *di.B) {
				b.Set(key.EcsCli, ecsMock)
				b.Set(key.Env, env)
				b.Set(key.Logger, test.NewLogger())
			}),
		}
		cm.taskArn = aws.String("task-arn")
		return ecsMock, cm
	}
	t.Run("should call ecscCli.StopTask and wait", func(t *testing.T) {
		ecsMock, cm := setup(t, test.DefaultEnvars())
		gomock.InOrder(
			ecsMock.EXPECT().StopTask(gomock.Any(), gomock.Any()).Return(&ecs.StopTaskOutput{}, nil),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("STOPPED")}},
			}, nil),
		)
		err := cm.stopTask(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should do nothing if task is not started", func(t *testing.T) {
		cm := &common{}
		err := cm.stopTask(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should error if StopTask failed", func(t *testing.T) {
		ecsMock, cm := setup(t, test.DefaultEnvars())
		ecsMock.EXPECT().StopTask(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
		err := cm.stopTask(context.TODO())
		assert.EqualError(t, err, "failed to stop canary task: error")
	})
	t.Run("should error wait time exceeds the limit", func(t *testing.T) {
		env := test.DefaultEnvars()
		env.CanaryTaskStoppedWait = 1
		ecsMock, cm := setup(t, env)
		gomock.InOrder(
			ecsMock.EXPECT().StopTask(gomock.Any(), gomock.Any()).Return(&ecs.StopTaskOutput{}, nil),
			ecsMock.EXPECT().DescribeTasks(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ecs.DescribeTasksOutput{
				Tasks: []ecstypes.Task{{LastStatus: aws.String("RUNNING")}},
			}, nil),
		)
		err := cm.stopTask(context.TODO())
		assert.ErrorContains(t, err, "failed to wait for canary task to be stopped")
	})
}
