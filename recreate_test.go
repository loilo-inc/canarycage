package cage_test

import (
	"context"
	"fmt"
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

func TestRecreate(t *testing.T) {
	setup := func(t *testing.T, passPhase int) (
		cage.Cage,
		*test.MockContext,
		*mock_awsiface.MockEcsClient,
		*gomock.Call,
	) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		m := mock_awsiface.NewMockEcsClient(ctrl)
		mocker := test.NewMockContext()
		mocker.CreateService(context.TODO(), env.ServiceDefinitionInput)
		phases := []func() *gomock.Call{
			func() *gomock.Call {
				// describe old service
				return m.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeServices)
			},
			func() *gomock.Call {
				// create next task definition
				return m.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).DoAndReturn(mocker.RegisterTaskDefinition)
			},
		}
		waiter := func() *gomock.Call {
			return m.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(mocker.DescribeServices)
		}
		swapPhase := []func() *gomock.Call{
			func() *gomock.Call {
				// create transition service
				return m.EXPECT().CreateService(gomock.Any(), gomock.Any()).DoAndReturn(mocker.CreateService)
			},
			// expect transition service to be ACTIVE
			waiter,
			func() *gomock.Call {
				// update transition service's desired count to old service's desired count
				return m.EXPECT().UpdateService(gomock.Any(), gomock.Any()).DoAndReturn(mocker.UpdateService)
			},
			// expect transition service to be ACTIVE
			waiter,
			func() *gomock.Call {
				// update old service's desired count to 0
				return m.EXPECT().UpdateService(gomock.Any(), gomock.Any()).DoAndReturn(mocker.UpdateService)
			},
			// expect old service to be ACTIVE
			waiter,
			func() *gomock.Call {
				// delete old service
				return m.EXPECT().DeleteService(gomock.Any(), gomock.Any()).DoAndReturn(mocker.DeleteService)
			},
			// expect old service to be INACTIVE
			waiter,
		}
		allPhases := append(phases, swapPhase...)
		allPhases = append(allPhases, swapPhase...)
		i := 0
		var prevCall *gomock.Call
		for {
			if i == passPhase || i == len(allPhases) {
				break
			}
			call := allPhases[i]()
			if prevCall != nil {
				call.After(prevCall)
			}
			prevCall = call
			i++
		}
		return cage.NewCage(&cage.Input{
			Env:     env,
			ECS:     m,
			ALB:     nil,
			EC2:     nil,
			Time:    test.NewFakeTime(),
			MaxWait: 1,
		}), mocker, m, prevCall
	}
	t.Run("basic", func(t *testing.T) {
		cagecli, mocker, _, _ := setup(t, -1)
		result, err := cagecli.Recreate(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, result.Service)
		assert.NotNil(t, result.TaskDefinition)
		assert.Equal(t, mocker.ActiveServiceSize(), 1)
		assert.Equal(t, mocker.RunningTaskSize(), 1)
		assert.Equal(t, len(mocker.TaskDefinitions.List()), 1)
		assert.Equal(t, *mocker.Services["service"].ServiceName, *result.Service.ServiceName)
		td := mocker.TaskDefinitions.List()[0]
		assert.Equal(t, *td.TaskDefinitionArn, *result.TaskDefinition.TaskDefinitionArn)
		assert.Equal(t, *mocker.Services["service"].TaskDefinition, *result.TaskDefinition.TaskDefinitionArn)
	})
	t.Run("should error if failed to describe old service", func(t *testing.T) {
		cagecli, _, ecsMock, _ := setup(t, 0)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
		result, err := cagecli.Recreate(context.Background())
		assert.EqualError(t, err, "couldn't describe service: error")
		assert.Nil(t, result)
	})
	t.Run("should error if old service doesn't exist", func(t *testing.T) {
		cagecli, _, ecsMock, _ := setup(t, 0)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Services: nil}, nil,
		)
		result, err := cagecli.Recreate(context.Background())
		assert.EqualError(t, err, "service 'service' does not exist. Use 'cage up' instead")
		assert.Nil(t, result)
	})
	t.Run("should error if old service is already INACTIVE", func(t *testing.T) {
		cagecli, _, ecsMock, _ := setup(t, 0)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Services: []ecstypes.Service{{Status: aws.String("INACTIVE")}}}, nil,
		)
		result, err := cagecli.Recreate(context.Background())
		assert.EqualError(t, err, "service 'service' is already INACTIVE. Use 'cage up' instead")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to create next task definition", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 1)
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.EqualError(t, err, "failed to register next task definition: error")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to create transition service", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 2)
		ecsMock.EXPECT().CreateService(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to create service")
		assert.Nil(t, result)
	})
	t.Run("should error if transition service is not ACTIVE", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 3)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Failures: []ecstypes.Failure{{Reason: aws.String("MISSING")}}}, nil,
		).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to wait for service")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to update transition service's desired count", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 4)
		ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to update service")
		assert.Nil(t, result)
	})
	t.Run("should error if transition service is not ACTIVE", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 5)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Failures: []ecstypes.Failure{{Reason: aws.String("MISSING")}}}, nil,
		).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to wait for service")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to update old service's desired count", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 6)
		ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to update service")
		assert.Nil(t, result)
	})
	t.Run("should error if old service is not ACTIVE", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 7)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Failures: []ecstypes.Failure{{Reason: aws.String("MISSING")}}}, nil,
		).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to wait for service")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to delete old service", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 8)
		ecsMock.EXPECT().DeleteService(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to delete service")
		assert.Nil(t, result)
	})
	t.Run("should error if old service is not INACTIVE", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 9)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Failures: []ecstypes.Failure{{Reason: aws.String("MISSING")}}}, nil,
		).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to wait for service")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to create new service", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 10)
		ecsMock.EXPECT().CreateService(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to create service")
		assert.Nil(t, result)
	})
	t.Run("should error if new service is not ACTIVE", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 11)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Failures: []ecstypes.Failure{{Reason: aws.String("MISSING")}}}, nil,
		).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to wait for service")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to update new service's desired count", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 12)
		ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to update service")
		assert.Nil(t, result)
	})
	t.Run("should error if old service is not ACTIVE", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 13)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Failures: []ecstypes.Failure{{Reason: aws.String("MISSING")}}}, nil,
		).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to wait for service")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to update transition service's desired count", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 14)
		ecsMock.EXPECT().UpdateService(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to update service")
		assert.Nil(t, result)
	})
	t.Run("should error if transition service is not ACTIVE", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 15)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Failures: []ecstypes.Failure{{Reason: aws.String("MISSING")}}}, nil,
		).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to wait for service")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to delete old service", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 16)
		ecsMock.EXPECT().DeleteService(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to delete service")
		assert.Nil(t, result)
	})
	t.Run("should error if old service is not INACTIVE", func(t *testing.T) {
		cagecli, _, ecsMock, call := setup(t, 17)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Failures: []ecstypes.Failure{{Reason: aws.String("MISSING")}}}, nil,
		).After(call)
		result, err := cagecli.Recreate(context.Background())
		assert.ErrorContains(t, err, "failed to wait for service")
		assert.Nil(t, result)
	})
}
