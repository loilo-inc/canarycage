package commands

import (
	"context"
	"testing"

	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/mocks/mock_types"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/canarycage/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSetupCage(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		envars := &env.Envars{Region: "us-west-2"}
		cageCli := mock_types.NewMockCage(gomock.NewController(t))
		cmd := NewCageCommands(func(ctx context.Context, envars *env.Envars) (types.Cage, error) {
			return cageCli, nil
		})
		v, err := cmd.setupCage(envars, "../../../fixtures")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, v, cageCli)
		assert.Equal(t, envars.Service, "service")
		assert.Equal(t, envars.Cluster, "cluster")
		assert.NotNil(t, envars.ServiceDefinitionInput)
		assert.NotNil(t, envars.TaskDefinitionInput)
	})
	t.Run("should skip load task definition if --taskDefinitionArn provided", func(t *testing.T) {
		envars := &env.Envars{Region: "us-west-2", TaskDefinitionArn: "arn"}
		cageCli := mock_types.NewMockCage(gomock.NewController(t))
		cmd := NewCageCommands(func(ctx context.Context, envars *env.Envars) (types.Cage, error) {
			return cageCli, nil
		})
		v, err := cmd.setupCage(envars, "../../../fixtures")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, v, cageCli)
		assert.Equal(t, envars.Service, "service")
		assert.Equal(t, envars.Cluster, "cluster")
		assert.NotNil(t, envars.ServiceDefinitionInput)
		assert.Nil(t, envars.TaskDefinitionInput)
	})
	t.Run("should error if error returned from NewCage", func(t *testing.T) {
		envars := &env.Envars{Region: "us-west-2"}
		cmd := NewCageCommands(func(ctx context.Context, envars *env.Envars) (types.Cage, error) {
			return nil, test.Err
		})
		_, err := cmd.setupCage(envars, "../../../fixtures")
		assert.EqualError(t, err, "error")
	})
}
