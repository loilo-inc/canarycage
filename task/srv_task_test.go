package task_test

import (
	"context"
	"testing"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/canarycage/timeout"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
)

func TestSrvTask(t *testing.T) {
	srvSvcName := "internal"
	srvNsName := "dev.local"
	registryArn := "arn://aaa/srv-" + srvSvcName
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
		b.Set(key.SrvCli, mocker.Srv)
		b.Set(key.TimeoutManager, timeout.NewManager(env, 1))
		b.Set(key.Time, test.NewFakeTime())
	})
	stask := task.NewSrvTask(d, &task.Input{
		TaskDefinition:       td.TaskDefinition,
		NetworkConfiguration: ecsSvc.Service.NetworkConfiguration,
	}, &ecstypes.ServiceRegistry{RegistryArn: &registryArn})
	_ = mocker.CreateSrvService(srvNsName, srvSvcName)
	err := stask.Start(ctx)
	assert.NoError(t, err)
	taskId := task.ArnToId(*stask.TaskArn())
	mocker.PutSrvInstHealth(taskId, "healthy")
	err = stask.Wait(ctx)
	assert.NoError(t, err)
	err = stask.Stop(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mocker.RunningTaskSize())
	assert.Equal(t, 0, len(mocker.SrvInsts[srvSvcName]))
}
