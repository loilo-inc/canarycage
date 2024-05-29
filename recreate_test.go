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
	t.Run("basic", func(t *testing.T) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		ctx := context.TODO()
		mocker, ecsMock, _, _ := test.Setup(ctrl, env, 1, "FARGATE")
		mocker.CreateService(ctx, env.ServiceDefinitionInput)
		cagecli := cage.NewCage(&cage.Input{
			Env:  env,
			ECS:  ecsMock,
			ALB:  nil,
			EC2:  nil,
			Time: test.NewFakeTime(),
		})
		result, err := cagecli.Recreate(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, result.Service)
		assert.NotNil(t, result.TaskDefinition)
		assert.Equal(t, mocker.ActiveServiceSize(), 1)
		assert.Equal(t, mocker.RunningTaskSize(), 1)
		assert.Equal(t, len(mocker.TaskDefinitions.List()), 2)
		assert.Equal(t, *mocker.Services["service"].ServiceName, *result.Service.ServiceName)
	})
	t.Run("should error if failed to describe old service", func(t *testing.T) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
		cagecli := cage.NewCage(&cage.Input{
			Env: env,
			ECS: ecsMock,
		})
		result, err := cagecli.Recreate(context.Background())
		assert.EqualError(t, err, "couldn't describe service: error")
		assert.Nil(t, result)
	})
	t.Run("should error if old service doesn't exist", func(t *testing.T) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Services: nil}, nil,
		)
		cagecli := cage.NewCage(&cage.Input{
			Env: env,
			ECS: ecsMock,
		})
		result, err := cagecli.Recreate(context.Background())
		assert.EqualError(t, err, "service 'service' does not exist. Use 'cage up' instead")
		assert.Nil(t, result)
	})
	t.Run("should error if old service is already INACTIVE", func(t *testing.T) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Services: []ecstypes.Service{{Status: aws.String("INACTIVE")}}}, nil,
		)
		cagecli := cage.NewCage(&cage.Input{
			Env: env,
			ECS: ecsMock,
		})
		result, err := cagecli.Recreate(context.Background())
		assert.EqualError(t, err, "service 'service' is already INACTIVE. Use 'cage up' instead")
		assert.Nil(t, result)
	})
	t.Run("should error if failed to create next task definition", func(t *testing.T) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(
			&ecs.DescribeServicesOutput{Services: []ecstypes.Service{{Status: aws.String("ACTIVE")}}}, nil,
		)
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
		cagecli := cage.NewCage(&cage.Input{
			Env: env,
			ECS: ecsMock,
		})
		result, err := cagecli.Recreate(context.Background())
		assert.EqualError(t, err, "failed to register next task definition: error")
		assert.Nil(t, result)
	})
}
