package task_test

import (
	"context"
	"testing"

	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/canarycage/timeout"
	"github.com/loilo-inc/canarycage/types"
	"github.com/stretchr/testify/assert"
)

func TestSimpleTask(t *testing.T) {
	ctx := context.TODO()
	mocker := test.NewMockContext()
	env := test.DefaultEnvars()
	td, _ := mocker.Ecs.RegisterTaskDefinition(ctx, env.TaskDefinitionInput)
	env.ServiceDefinitionInput.TaskDefinition = td.TaskDefinition.TaskDefinitionArn
	ecsSvc, _ := mocker.Ecs.CreateService(ctx, env.ServiceDefinitionInput)
	stask := task.NewSimpleTask(&task.Input{
		Deps: &types.Deps{
			Env:  env,
			Ecs:  mocker.Ecs,
			Ec2:  mocker.Ec2,
			Alb:  mocker.Alb,
			Srv:  mocker.Srv,
			Time: test.NewFakeTime(),
		},
		TaskDefinition:       td.TaskDefinition,
		NetworkConfiguration: ecsSvc.Service.NetworkConfiguration,
		Timeout:              timeout.NewManager(1, &timeout.Input{}),
	})
	err := stask.Start(ctx)
	assert.NoError(t, err)
	err = stask.Wait(ctx)
	assert.NoError(t, err)
	err = stask.Stop(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mocker.RunningTaskSize())
}
