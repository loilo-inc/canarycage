package cage_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	cage "github.com/loilo-inc/canarycage"
	"github.com/loilo-inc/canarycage/test"
	"github.com/stretchr/testify/assert"
)

func TestCage_Up(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		ctx, ecsMock, _, _ := test.Setup(ctrl, env, 1, "FARGATE")
		delete(ctx.Services, env.Service)
		cagecli := cage.NewCage(&cage.Input{
			Env: env,
			Ecs: ecsMock,
		})
		result, err := cagecli.Up(context.Background())
		assert.Nil(t, err)
		assert.NotNil(t, result.Service)
		assert.NotNil(t, result.TaskDefinition)
	})
	t.Run("should show error if service exists", func(t *testing.T) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		_, ecsMock, _, _ := test.Setup(ctrl, env, 1, "FARGATE")
		cagecli := cage.NewCage(&cage.Input{
			Env: env,
			Ecs: ecsMock,
		})
		result, err := cagecli.Up(context.Background())
		assert.Nil(t, result)
		assert.EqualError(t, err, "service 'service' already exists. Use 'cage rollout' instead")
	})
}
