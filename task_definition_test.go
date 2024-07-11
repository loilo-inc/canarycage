package cage

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/xerrors"
)

func TestCage_CreateNextTaskDefinition(t *testing.T) {
	t.Run("should return task definition if taskDefinitionArn is set", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		env := &env.Envars{
			TaskDefinitionArn: "arn://aaa",
		}
		c := &cage{
			di: di.NewDomain(func(b *di.B) {
				b.Set(key.Env, env)
				b.Set(key.EcsCli, ecsMock)
			})}
		ecsMock.EXPECT().DescribeTaskDefinition(gomock.Any(), gomock.Any()).Return(&ecs.DescribeTaskDefinitionOutput{
			TaskDefinition: &ecstypes.TaskDefinition{},
		}, nil)
		td, err := c.CreateNextTaskDefinition(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, td)
	})
	t.Run("should return error if taskDefinitionArn is set and failed to describe task definition", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		env := &env.Envars{
			TaskDefinitionArn: "arn://aaa",
		}
		c := &cage{
			di: di.NewDomain(func(b *di.B) {
				b.Set(key.Env, env)
				b.Set(key.EcsCli, ecsMock)
			})}
		ecsMock.EXPECT().DescribeTaskDefinition(gomock.Any(), gomock.Any()).Return(nil, xerrors.New("error"))
		td, err := c.CreateNextTaskDefinition(context.Background())
		assert.Errorf(t, err, "failed to describe next task definition: error")
		assert.Nil(t, td)
	})
	t.Run("should return task definition if taskDefinitionArn is not set", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		env := test.DefaultEnvars()
		c := &cage{
			di: di.NewDomain(func(b *di.B) {
				b.Set(key.Env, env)
				b.Set(key.EcsCli, ecsMock)
			})}
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).Return(&ecs.RegisterTaskDefinitionOutput{
			TaskDefinition: &ecstypes.TaskDefinition{
				Family:   env.TaskDefinitionInput.Family,
				Revision: 1,
			},
		}, nil)
		td, err := c.CreateNextTaskDefinition(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, td)
	})
	t.Run("should return error if taskDefinitionArn is not set and failed to register task definition", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ecsMock := mock_awsiface.NewMockEcsClient(ctrl)
		env := test.DefaultEnvars()
		c := &cage{
			di: di.NewDomain(func(b *di.B) {
				b.Set(key.Env, env)
				b.Set(key.EcsCli, ecsMock)
			})}
		ecsMock.EXPECT().RegisterTaskDefinition(gomock.Any(), gomock.Any()).Return(nil, xerrors.New("error"))
		td, err := c.CreateNextTaskDefinition(context.Background())
		assert.Errorf(t, err, "failed to register next task definition: error")
		assert.Nil(t, td)
	})
}
