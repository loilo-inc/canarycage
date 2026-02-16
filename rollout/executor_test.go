package rollout

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/v5/env"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/v5/mocks/mock_task"
	"github.com/loilo-inc/canarycage/v5/task"
	"github.com/loilo-inc/canarycage/v5/test"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewExecutor(t *testing.T) {
	td := &ecstypes.TaskDefinition{}
	di := di.EmptyDomain()
	e := NewExecutor(di, td)
	v, ok := e.(*executor)
	assert.True(t, ok)
	assert.Equal(t, td, v.td)
	assert.Equal(t, di, v.di)
}

func TestExecutor_Rollout(t *testing.T) {
	setup := func(t *testing.T) (
		*executor,
		*env.Envars,
		*test.MockContext,
		*mock_awsiface.MockEcsClient,
		*mock_task.MockTask,
		*ecstypes.TaskDefinition,
	) {
		ctrl := gomock.NewController(t)
		envars := test.DefaultEnvars()
		factoryMock := mock_task.NewMockFactory(ctrl)
		taskMock := mock_task.NewMockTask(ctrl)
		mocker := test.NewMockContext()
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), envars.TaskDefinitionInput)
		envars.ServiceDefinitionInput.TaskDefinition = td.TaskDefinition.TaskDefinitionArn
		srv, _ := mocker.Ecs.CreateService(context.TODO(), envars.ServiceDefinitionInput)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Env, envars)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.TaskFactory, factoryMock)
			b.Set(key.Logger, test.NewLogger())
		})
		factoryMock.EXPECT().NewAlbTask(&task.Input{
			TaskDefinition:       td.TaskDefinition,
			NetworkConfiguration: srv.Service.NetworkConfiguration,
			PlatformVersion:      srv.Service.PlatformVersion,
		}, &srv.Service.LoadBalancers[0]).Return(taskMock)
		e := &executor{di: d, td: td.TaskDefinition}
		return e, envars, mocker, ecsMock, taskMock, td.TaskDefinition
	}
	t.Run("basic", func(t *testing.T) {
		e, envars, mocker, ecsMock, taskMock, td := setup(t)
		gomock.InOrder(
			ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(mocker.Ecs.DescribeServices),
			taskMock.EXPECT().Start(gomock.Any()).Return(nil),
			taskMock.EXPECT().Wait(gomock.Any()).Return(nil),
			ecsMock.EXPECT().UpdateService(gomock.Any(), &ecs.UpdateServiceInput{
				Cluster:                     &envars.Cluster,
				Service:                     &envars.Service,
				TaskDefinition:              td.TaskDefinitionArn,
				ServiceConnectConfiguration: nil,
				LoadBalancers:               nil,
				NetworkConfiguration:        nil,
				PlatformVersion:             nil,
				VolumeConfigurations:        nil,
				ServiceRegistries:           nil,
			}).
				DoAndReturn(mocker.Ecs.UpdateService),
			ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(mocker.Ecs.DescribeServices),
			taskMock.EXPECT().Stop(gomock.Any()).Return(nil),
		)
		err := e.RollOut(context.TODO(), &types.RollOutInput{})
		if err != nil {
			t.Errorf("RollOut() error = %v", err)
		}
		srv, _ := mocker.GetEcsService(envars.Service)
		assert.Equal(t, *srv.TaskDefinition, *td.TaskDefinitionArn)
		assert.True(t, e.ServiceUpdated())
	})
	t.Run("updateService", func(t *testing.T) {
		e, envars, mocker, ecsMock, taskMock, td := setup(t)
		gomock.InOrder(
			taskMock.EXPECT().Start(gomock.Any()).Return(nil),
			taskMock.EXPECT().Wait(gomock.Any()).Return(nil),
			ecsMock.EXPECT().UpdateService(gomock.Any(), &ecs.UpdateServiceInput{
				Cluster:                     &envars.Cluster,
				Service:                     &envars.Service,
				TaskDefinition:              td.TaskDefinitionArn,
				ServiceConnectConfiguration: envars.ServiceDefinitionInput.ServiceConnectConfiguration,
				LoadBalancers:               envars.ServiceDefinitionInput.LoadBalancers,
				NetworkConfiguration:        envars.ServiceDefinitionInput.NetworkConfiguration,
				PlatformVersion:             envars.ServiceDefinitionInput.PlatformVersion,
				VolumeConfigurations:        envars.ServiceDefinitionInput.VolumeConfigurations,
				ServiceRegistries:           envars.ServiceDefinitionInput.ServiceRegistries,
			}).
				DoAndReturn(mocker.Ecs.UpdateService),
			ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(mocker.Ecs.DescribeServices),
			taskMock.EXPECT().Stop(gomock.Any()).Return(nil),
		)
		err := e.RollOut(context.TODO(), &types.RollOutInput{UpdateService: true})
		if err != nil {
			t.Errorf("RollOut() error = %v", err)
		}
		srv, _ := mocker.GetEcsService(envars.Service)
		assert.Equal(t, *srv.TaskDefinition, *td.TaskDefinitionArn)
		assert.True(t, e.ServiceUpdated())
	})
}

func TestExecutor_Rollout_Failure(t *testing.T) {
	setup := func(t *testing.T) (*executor, *env.Envars, *mock_awsiface.MockEcsClient, *mock_task.MockFactory, *mock_task.MockTask) {
		ctrl := gomock.NewController(t)
		envars := test.DefaultEnvars()
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		taskMock := mock_task.NewMockTask(ctrl)
		factoryMock := mock_task.NewMockFactory(ctrl)
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Env, envars)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.TaskFactory, factoryMock)
			b.Set(key.Logger, test.NewLogger())
		})
		td := &ecstypes.TaskDefinition{
			TaskDefinitionArn: aws.String("arn://aaa"),
			Family:            aws.String("family"),
			Revision:          1,
		}
		e := &executor{di: d, td: td}
		return e, envars, ecsMock, factoryMock, taskMock
	}
	t.Run("should not call task.Task.Stop() if task not created", func(t *testing.T) {
		e, _, ecsCli, _, _ := setup(t)
		ecsCli.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(nil, test.Err)
		err := e.RollOut(context.TODO(), &types.RollOutInput{})
		assert.EqualError(t, err, "error")
		assert.False(t, e.ServiceUpdated())
	})
	t.Run("should call task.Task.Stop() even if task.Task.Start() failed", func(t *testing.T) {
		e, _, _, factoryMock, taskMock := setup(t)
		gomock.InOrder(
			factoryMock.EXPECT().NewAlbTask(gomock.Any(), gomock.Any()).Return(taskMock),
			taskMock.EXPECT().Start(gomock.Any()).Return(test.Err),
			taskMock.EXPECT().Stop(gomock.Any()).Return(nil),
		)
		err := e.RollOut(context.TODO(), &types.RollOutInput{UpdateService: true})
		assert.EqualError(t, err, "error")
		assert.False(t, e.ServiceUpdated())
	})
	t.Run("should call task.Task.Stop() even if task.Task.Wait() failed", func(t *testing.T) {
		e, _, _, factoryMock, taskMock := setup(t)
		gomock.InOrder(
			factoryMock.EXPECT().NewAlbTask(gomock.Any(), gomock.Any()).Return(taskMock),
			taskMock.EXPECT().Start(gomock.Any()).Return(nil),
			taskMock.EXPECT().Wait(gomock.Any()).Return(test.Err),
			taskMock.EXPECT().Stop(gomock.Any()).Return(nil),
		)
		err := e.RollOut(context.TODO(), &types.RollOutInput{UpdateService: true})
		assert.EqualError(t, err, "error")
		assert.False(t, e.ServiceUpdated())
	})
	t.Run("should call task.Task.Stop() even if ecs.UpdateService() failed", func(t *testing.T) {
		e, _, ecsMock, factoryMock, taskMock := setup(t)
		gomock.InOrder(
			factoryMock.EXPECT().NewAlbTask(gomock.Any(), gomock.Any()).Return(taskMock),
			taskMock.EXPECT().Start(gomock.Any()).Return(nil),
			taskMock.EXPECT().Wait(gomock.Any()).Return(nil),
			ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any()).
				Return(nil, test.Err),
			taskMock.EXPECT().Stop(gomock.Any()).Return(nil),
		)
		err := e.RollOut(context.TODO(), &types.RollOutInput{UpdateService: true})
		assert.EqualError(t, err, "error")
		assert.False(t, e.ServiceUpdated())
	})
	t.Run("should call task.Task.Stop() even if ecs.NewServicesStableWaiter.Wait() failed", func(t *testing.T) {
		e, _, ecsMock, factoryMock, taskMock := setup(t)
		gomock.InOrder(
			factoryMock.EXPECT().NewAlbTask(gomock.Any(), gomock.Any()).Return(taskMock),
			taskMock.EXPECT().Start(gomock.Any()).Return(nil),
			taskMock.EXPECT().Wait(gomock.Any()).Return(nil),
			ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any()).
				Return(&ecs.UpdateServiceOutput{}, nil),
			ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&ecs.DescribeServicesOutput{
					Services: []ecstypes.Service{{Status: aws.String("INACTIVE")}},
				}, nil),
			taskMock.EXPECT().Stop(gomock.Any()).Return(nil),
		)
		err := e.RollOut(context.TODO(), &types.RollOutInput{UpdateService: true})
		assert.EqualError(t, err, "waiter state transitioned to Failure")
		assert.True(t, e.ServiceUpdated())
	})
	t.Run("should log error if task.Task.Stop() failed", func(t *testing.T) {
		e, _, ecsMock, factoryMock, taskMock := setup(t)
		gomock.InOrder(
			factoryMock.EXPECT().NewAlbTask(gomock.Any(), gomock.Any()).Return(taskMock),
			taskMock.EXPECT().Start(gomock.Any()).Return(nil),
			taskMock.EXPECT().Wait(gomock.Any()).Return(nil),
			ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any()).
				Return(&ecs.UpdateServiceOutput{}, nil),
			ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&ecs.DescribeServicesOutput{
					Services: []ecstypes.Service{{Status: aws.String("INACTIVE")}},
				}, nil),
			taskMock.EXPECT().Stop(gomock.Any()).Return(test.Err),
		)
		err := e.RollOut(context.TODO(), &types.RollOutInput{UpdateService: true})
		assert.EqualError(t, err, "error")
		assert.True(t, e.ServiceUpdated())
	})
}
