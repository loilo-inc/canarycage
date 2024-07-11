package cage_test

import (
	"context"
	"testing"

	cage "github.com/loilo-inc/canarycage"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCage_Up(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		ctx, ecsMock, _, _ := test.Setup(ctrl, env, 1, "FARGATE")
		delete(ctx.Services, env.Service)
		cagecli := cage.NewCage(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
		}))
		result, err := cagecli.Up(context.Background())
		assert.Nil(t, err)
		assert.NotNil(t, result.Service)
		assert.NotNil(t, result.TaskDefinition)
	})
	t.Run("should show error if service exists", func(t *testing.T) {
		env := test.DefaultEnvars()
		ctrl := gomock.NewController(t)
		_, ecsMock, _, _ := test.Setup(ctrl, env, 1, "FARGATE")
		cagecli := cage.NewCage(di.NewDomain(func(b *di.B) {
			b.Set(key.Env, env)
			b.Set(key.EcsCli, ecsMock)
		}))
		result, err := cagecli.Up(context.Background())
		assert.Nil(t, result)
		assert.EqualError(t, err, "service 'service' already exists. Use 'cage rollout' instead")
	})
}
