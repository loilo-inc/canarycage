package cage

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	alb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	albtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/loilo-inc/canarycage/v5/env"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/v5/task"
	"github.com/loilo-inc/canarycage/v5/test"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// fake integration test with test.MockContext
func TestCage_RollOut(t *testing.T) {
	for i := 0b000; i < 0b1000; i++ {
		env := test.DefaultEnvars()
		isEc2 := i&0b001 > 0
		useExistingTd := i&0b010 > 0
		updateService := i&0b100 > 0
		if isEc2 {
			env.CanaryInstanceArn = "arn:aws:ecs:us-west-2:123456789012:container-instance/123456789012"
		}
		if useExistingTd {
			env.TaskDefinitionArn = "arn:aws:ecs:us-west-2:123456789012:task-definition/td"
			env.TaskDefinitionInput = nil
		}
		for j := 0; j < 3; j++ {
			t.Run(fmt.Sprintf("isEc2=%t, useTd=%t, lbcount=%d", isEc2, useExistingTd, j), func(t *testing.T) {
				integrationTest(t, env, j, &types.RollOutInput{UpdateService: updateService})
			})
		}
	}
}

func integrationTest(t *testing.T, env *env.Envars, lbcount int, input *types.RollOutInput) {
	mocker := test.NewMockContext()
	var td *ecstypes.TaskDefinition
	if env.TaskDefinitionArn == "" {
		o, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), env.TaskDefinitionInput)
		td = o.TaskDefinition
	} else {
		o, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), test.DefaultEnvars().TaskDefinitionInput)
		td = o.TaskDefinition
		env.TaskDefinitionArn = *td.TaskDefinitionArn
	}
	env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinitionArn
	_, _ = mocker.Ecs.CreateService(context.TODO(), env.ServiceDefinitionInput)
	var lbs []ecstypes.LoadBalancer
	for i := 0; i < lbcount; i++ {
		tgArn := aws.String(fmt.Sprintf("tg%d", i+1))
		lbs = append(lbs, ecstypes.LoadBalancer{
			ContainerName:  aws.String("container"),
			ContainerPort:  aws.Int32(80),
			TargetGroupArn: tgArn,
		})
		mocker.Alb.RegisterTargets(context.TODO(), &alb.RegisterTargetsInput{
			TargetGroupArn: tgArn,
			Targets: []albtypes.TargetDescription{
				{Id: aws.String(fmt.Sprintf("127.0.0.%d", i+1))},
			},
		})
	}
	c := &cage{di: di.NewDomain(func(b *di.B) {
		b.Set(key.Env, env)
		b.Set(key.Ec2Cli, mocker.Ec2)
		b.Set(key.EcsCli, mocker.Ecs)
		b.Set(key.AlbCli, mocker.Alb)
		b.Set(key.Logger, test.NewLogger())
		b.Set(key.Time, test.NewFakeTime())
		b.Set(key.TaskFactory, task.NewFactory(b.Future()))
	})}
	assert.Equal(t, 1, mocker.RunningTaskSize())
	assert.Equal(t, 1, len(mocker.TaskDefinitions.List()))
	for _, lb := range lbs {
		assert.Equal(t, 1, len(mocker.TargetGroups[*lb.TargetGroupArn].Targets))
	}
	result, err := c.RollOut(context.TODO(), input)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, result.ServiceUpdated)
	assert.Equal(t, 1, mocker.RunningTaskSize())
	assert.Equal(t, 1, len(mocker.Services))
	if env.TaskDefinitionArn != "" {
		assert.Equal(t, 1, len(mocker.TaskDefinitions.List()))
	} else {
		assert.Equal(t, 2, len(mocker.TaskDefinitions.List()))
	}
	for _, lb := range lbs {
		assert.Equal(t, 1, len(mocker.TargetGroups[*lb.TargetGroupArn].Targets))
	}
	updatedService, _ := mocker.GetEcsService(env.Service)
	if env.TaskDefinitionArn != "" {
		assert.Equal(t, env.TaskDefinitionArn, *updatedService.TaskDefinition)
	} else {
		assert.NotEqual(t, *td.TaskDefinitionArn, *updatedService.TaskDefinition)
	}
}

func TestCage_Rollout_Failure(t *testing.T) {
	t.Run("should error if DescribeServices failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		env := test.DefaultEnvars()
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		c := &cage{di: di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Logger, test.NewLogger())
		})}
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(nil, test.Err)
		result, err := c.RollOut(context.TODO(), &types.RollOutInput{})
		assert.EqualError(t, err, "failed to describe current service due to: error")
		assert.False(t, result.ServiceUpdated)
	})
	t.Run("should error if service doesn't exist", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		env := test.DefaultEnvars()
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		c := &cage{di: di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Logger, test.NewLogger())
		})}
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(&ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{},
		}, nil)
		result, err := c.RollOut(context.TODO(), &types.RollOutInput{})
		assert.ErrorContains(t, err, "service 'service' doesn't exist")
		assert.False(t, result.ServiceUpdated)
	})
	t.Run("should error if service status is not ACTIVE", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		env := test.DefaultEnvars()
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		c := &cage{di: di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Logger, test.NewLogger())
		})}
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(&ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{{
				ServiceName: aws.String("service"),
				Status:      aws.String("INACTIVE")}},
		}, nil)
		result, err := c.RollOut(context.TODO(), &types.RollOutInput{})
		assert.ErrorContains(t, err, "😵 service 'service' status is 'INACTIVE'")
		assert.False(t, result.ServiceUpdated)
	})
	t.Run("should error if LaunchType is EC2 and --canaryInstanceArn is not provided", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		env := test.DefaultEnvars()
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		c := &cage{di: di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Logger, test.NewLogger())
		})}
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(&ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{{
				ServiceName: aws.String("service"),
				Status:      aws.String("ACTIVE"),
				LaunchType:  ecstypes.LaunchTypeEc2}},
		}, nil)
		result, err := c.RollOut(context.TODO(), &types.RollOutInput{})
		assert.ErrorContains(t, err, "--canaryInstanceArn is required when LaunchType = 'EC2'")
		assert.False(t, result.ServiceUpdated)
	})
	t.Run("should error if CreateNextTaskDefinition failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		env := test.DefaultEnvars()
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		c := &cage{di: di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
			b.Set(key.Logger, test.NewLogger())
		})}
		ecsMock.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Return(&ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{{
				ServiceName: aws.String("service"),
				Status:      aws.String("ACTIVE")}},
		}, nil)
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).Return(nil, test.Err)
		result, err := c.RollOut(context.TODO(), &types.RollOutInput{})
		assert.EqualError(t, err, "failed to register next task definition due to: error")
		assert.False(t, result.ServiceUpdated)
	})
}
