package cage_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	cage "github.com/loilo-inc/canarycage"
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
		assert.Equal(t, len(mocker.Services), 1)
		assert.Equal(t, len(mocker.TaskDefinitions.List()), 2)
		assert.Equal(t, *mocker.Services["service"].ServiceName, *result.Service.ServiceName)
	})
}
