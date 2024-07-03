package task_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
)

func TestAlbTask(t *testing.T) {
	setup := func(env *env.Envars) (task.Task, *test.MockContext) {
		mocker := test.NewMockContext()
		ctx := context.TODO()
		td, _ := mocker.Ecs.RegisterTaskDefinition(ctx, env.TaskDefinitionInput)
		env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinition.TaskDefinitionArn
		ecsSvc, _ := mocker.Ecs.CreateService(ctx, env.ServiceDefinitionInput)
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, mocker.Ecs)
			b.Set(key.Ec2Cli, mocker.Ec2)
			b.Set(key.AlbCli, mocker.Alb)
			b.Set(key.Time, test.NewFakeTime())
		})
		stask := task.NewAlbTask(d, &task.Input{
			TaskDefinition:       td.TaskDefinition,
			NetworkConfiguration: ecsSvc.Service.NetworkConfiguration,
		}, &ecsSvc.Service.LoadBalancers[0])
		mocker.Alb.RegisterTargets(ctx, &elbv2.RegisterTargetsInput{
			TargetGroupArn: ecsSvc.Service.LoadBalancers[0].TargetGroupArn,
		})
		return stask, mocker
	}
	t.Run("fargate", func(t *testing.T) {
		env := test.DefaultEnvars()
		stask, mocker := setup(env)
		ctx := context.TODO()
		err := stask.Start(ctx)
		assert.NoError(t, err)
		err = stask.Wait(ctx)
		assert.NoError(t, err)
		err = stask.Stop(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, mocker.RunningTaskSize())
	})
	t.Run("ec2", func(t *testing.T) {
		env := test.DefaultEnvars()
		env.CanaryInstanceArn = "arn://ec2"
		stask, mocker := setup(env)
		ctx := context.TODO()
		err := stask.Start(ctx)
		assert.NoError(t, err)
		err = stask.Wait(ctx)
		assert.NoError(t, err)
		err = stask.Stop(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, mocker.RunningTaskSize())
	})
}

func TestAlbTask_WaitUntilTargetHealthy(t *testing.T) {
	target := &elbv2types.TargetDescription{
		Id:               aws.String("127.0.0.1"),
		Port:             aws.Int32(80),
		AvailabilityZone: aws.String("ap-northeast-1a"),
	}
	setup := func(t *testing.T) (*mock_awsiface.MockAlbClient, *task.AlbTaskExport) {
		ctrl := gomock.NewController(t)
		env := test.DefaultEnvars()
		mocker := test.NewMockContext()
		albMock := mock_awsiface.NewMockAlbClient(ctrl)
		td, _ := mocker.Ecs.RegisterTaskDefinition(context.TODO(), env.TaskDefinitionInput)
		atask := task.NewAlbTaskExport(di.NewDomain(func(b *di.B) {
			b.Set(key.AlbCli, albMock)
			b.Set(key.Time, test.NewFakeTime())
		}), &task.Input{
			TaskDefinition:       td.TaskDefinition,
			NetworkConfiguration: env.ServiceDefinitionInput.NetworkConfiguration,
		}, &env.ServiceDefinitionInput.LoadBalancers[0])
		atask.TaskArn = aws.String("arn://task")
		atask.Target = target
		return albMock, atask
	}
	t.Run("should call DescribeTargetHealth periodically", func(t *testing.T) {
		albMock, atask := setup(t)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumUnused},
						Target: target,
					},
				},
			}, nil).Times(1),
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumHealthy},
						Target: target,
					},
				},
			}, nil).Times(1),
		)
		err := atask.WaitUntilTargetHealthy(context.TODO())
		assert.NoError(t, err)
	})
	t.Run("should error if DescribeTargetHealth failed", func(t *testing.T) {
		albMock, atask := setup(t)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(nil, assert.AnError).Times(1),
		)
		err := atask.WaitUntilTargetHealthy(context.TODO())
		assert.EqualError(t, err, assert.AnError.Error())
	})
	t.Run("should error if context is canceled", func(t *testing.T) {
		_, atask := setup(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := atask.WaitUntilTargetHealthy(ctx)
		assert.EqualError(t, err, "context canceled")
	})
	t.Run("should error if target is not registered", func(t *testing.T) {
		albMock, atask := setup(t)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{},
			}, nil).Times(1),
		)
		err := atask.WaitUntilTargetHealthy(context.TODO())
		assert.EqualError(t, err, fmt.Sprintf(
			"'%s' is not registered to the target group '%s'", *target.Id, *atask.Lb.TargetGroupArn),
		)
	})
	t.Run("should error if target unhelthy counts exceed the limit", func(t *testing.T) {
		albMock, atask := setup(t)
		gomock.InOrder(
			albMock.EXPECT().DescribeTargetHealth(gomock.Any(), gomock.Any()).Return(&elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumUnhealthy},
						Target: target,
					},
				},
			}, nil).Times(5),
		)
		err := atask.WaitUntilTargetHealthy(context.TODO())
		assert.EqualError(t, err, fmt.Sprintf(
			"canary task '%s' (%s:%d) hasn't become to be healthy. The most recent state: %s",
			*atask.TaskArn, *target.Id, *target.Port, elbv2types.TargetHealthStateEnumUnhealthy,
		),
		)
	})
}
