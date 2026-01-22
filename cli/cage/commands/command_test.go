package commands

import (
	"context"
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/mocks/mock_types"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/canarycage/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSetupCage(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"
		cageCli := mock_types.NewMockCage(gomock.NewController(t))
		cmd := NewCageCommands(func(ctx context.Context, envars *cageapp.CageCmdInput) (types.Cage, error) {
			return cageCli, nil
		})
		v, err := cmd.setupCage(input, "../../../fixtures")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, v, cageCli)
		assert.Equal(t, input.Service, "service")
		assert.Equal(t, input.Cluster, "cluster")
		assert.NotNil(t, input.ServiceDefinitionInput)
		assert.NotNil(t, input.TaskDefinitionInput)
	})
	t.Run("should skip load task definition if --taskDefinitionArn provided", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"
		input.TaskDefinitionArn = "arn"
		cageCli := mock_types.NewMockCage(gomock.NewController(t))
		cmd := NewCageCommands(func(ctx context.Context, input *cageapp.CageCmdInput) (types.Cage, error) {
			return cageCli, nil
		})
		v, err := cmd.setupCage(input, "../../../fixtures")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, v, cageCli)
		assert.Equal(t, input.Service, "service")
		assert.Equal(t, input.Cluster, "cluster")
		assert.NotNil(t, input.ServiceDefinitionInput)
		assert.Nil(t, input.TaskDefinitionInput)
	})
	t.Run("should error if error returned from NewCage", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"
		cmd := NewCageCommands(func(ctx context.Context, input *cageapp.CageCmdInput) (types.Cage, error) {
			return nil, test.Err
		})
		_, err := cmd.setupCage(input, "../../../fixtures")
		assert.EqualError(t, err, "error")
	})
}
