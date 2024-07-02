package task_test

import (
	"context"
	"testing"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
)

func TestAlbTask(t *testing.T) {
	mocker := test.NewMockContext()
	env := test.DefaultEnvars()
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
	err := stask.Start(ctx)
	assert.NoError(t, err)
	err = stask.Wait(ctx)
	assert.NoError(t, err)
	err = stask.Stop(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mocker.RunningTaskSize())
}
