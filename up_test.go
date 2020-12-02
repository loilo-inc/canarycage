package cage

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCage_Up(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		env := DefaultEnvars()
		ctrl := gomock.NewController(t)
		ctx, ecsMock, _, _ := Setup(ctrl, env, 1, "FARGATE")
		delete(ctx.Services, env.Service)
		cagecli := NewCage(&Input{
			Env: env,
			ECS: ecsMock,
			ALB: nil,
			EC2: nil,
		})
		result, err := cagecli.Up(context.Background())
		assert.Nil(t, err)
		assert.NotNil(t, result.Service)
		assert.NotNil(t, result.TaskDefinition)
	})
	t.Run("should show error if service exists", func(t *testing.T) {
		env := DefaultEnvars()
		ctrl := gomock.NewController(t)
		_, ecsMock, _, _ := Setup(ctrl, env, 1, "FARGATE")
		cagecli := NewCage(&Input{
			Env: env,
			ECS: ecsMock,
			ALB: nil,
			EC2: nil,
		})
		result, err := cagecli.Up(context.Background())
		assert.Nil(t, result)
		assert.EqualError(t, err, "service 'service' already exists. Use 'cage rollout' instead")
	})
}
