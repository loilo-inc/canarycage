package commands

import (
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/stretchr/testify/assert"
)

func TestSetupCage(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"
		err := setupCage(t.Context(), input, "../../../fixtures")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, input.Service, "service")
		assert.Equal(t, input.Cluster, "cluster")
		assert.NotNil(t, input.ServiceDefinitionInput)
		assert.NotNil(t, input.TaskDefinitionInput)
	})
	t.Run("should skip load task definition if --taskDefinitionArn provided", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"
		input.TaskDefinitionArn = "arn"
		err := setupCage(t.Context(), input, "../../../fixtures")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, input.Service, "service")
		assert.Equal(t, input.Cluster, "cluster")
		assert.NotNil(t, input.ServiceDefinitionInput)
		assert.Nil(t, input.TaskDefinitionInput)
	})
}
