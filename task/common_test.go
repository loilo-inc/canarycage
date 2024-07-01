package task_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
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
			cm := task.NewCommonExport(di.NewDomain(func(b *di.B) {
				b.Set(key.Env, envars)
				b.Set(key.EcsCli, ecsMock)
			}), &task.Input{TaskDefinition: td})
			err := cm.Start(context.TODO())
			assert.NoError(t, err)
			assert.Equal(t, "task-arn", *cm.TaskArn())
		})
		t.Run("should error if task failed to start", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			td := &ecstypes.TaskDefinition{}
			ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
			ecsMock.EXPECT().RunTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
			envars := test.DefaultEnvars()
			cm := task.NewCommonExport(di.NewDomain(func(b *di.B) {
				b.Set(key.Env, envars)
				b.Set(key.EcsCli, ecsMock)
			}), &task.Input{TaskDefinition: td})
			err := cm.Start(context.TODO())
			assert.EqualError(t, err, "error")
			assert.Nil(t, cm.TaskArn())
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
			cm := task.NewCommonExport(di.NewDomain(func(b *di.B) {
				b.Set(key.Env, envars)
				b.Set(key.EcsCli, ecsMock)
			}), &task.Input{TaskDefinition: td})
			err := cm.Start(context.TODO())
			assert.NoError(t, err)
			assert.Equal(t, "task-arn", *cm.TaskArn())
		})
		t.Run("should error if task failed to start", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			td := &ecstypes.TaskDefinition{}
			ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
			ecsMock.EXPECT().StartTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
			envars := test.DefaultEnvars()
			envars.CanaryInstanceArn = "instance-arn"
			cm := task.NewCommonExport(di.NewDomain(func(b *di.B) {
				b.Set(key.Env, envars)
				b.Set(key.EcsCli, ecsMock)
			}), &task.Input{TaskDefinition: td})
			err := cm.Start(context.TODO())
			assert.EqualError(t, err, "error")
			assert.Nil(t, cm.TaskArn())
		})
	})
}
