package rollout

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
)

func TestApplyServiceDefinitionToUpdateInput(t *testing.T) {
	t.Run("preserves nil slice fields", func(t *testing.T) {
		updateInput := &ecs.UpdateServiceInput{}
		applyServiceDefinitionToUpdateInput(updateInput, &ecs.CreateServiceInput{})

		assert.Nil(t, updateInput.CapacityProviderStrategy)
		assert.Nil(t, updateInput.LoadBalancers)
		assert.Nil(t, updateInput.ServiceRegistries)
		assert.Nil(t, updateInput.VolumeConfigurations)
		assert.Nil(t, updateInput.PlacementConstraints)
		assert.Nil(t, updateInput.PlacementStrategy)
		assert.Nil(t, updateInput.VpcLatticeConfigurations)

		assert.Nil(t, updateInput.NetworkConfiguration)
		assert.Nil(t, updateInput.ServiceConnectConfiguration)
		assert.Nil(t, updateInput.PlatformVersion)
		assert.Nil(t, updateInput.DeploymentConfiguration)
		assert.Nil(t, updateInput.HealthCheckGracePeriodSeconds)
		assert.Equal(t, ecstypes.PropagateTags(""), updateInput.PropagateTags)
	})

	t.Run("preserves explicit empty slice fields", func(t *testing.T) {
		serviceInput := &ecs.CreateServiceInput{
			CapacityProviderStrategy: []ecstypes.CapacityProviderStrategyItem{},
			LoadBalancers:            []ecstypes.LoadBalancer{},
			ServiceRegistries:        []ecstypes.ServiceRegistry{},
			VolumeConfigurations:     []ecstypes.ServiceVolumeConfiguration{},
			PlacementConstraints:     []ecstypes.PlacementConstraint{},
			PlacementStrategy:        []ecstypes.PlacementStrategy{},
			VpcLatticeConfigurations: []ecstypes.VpcLatticeConfiguration{},
		}
		updateInput := &ecs.UpdateServiceInput{}
		applyServiceDefinitionToUpdateInput(updateInput, serviceInput)

		assert.Equal(t, serviceInput.CapacityProviderStrategy, updateInput.CapacityProviderStrategy)
		assert.Equal(t, serviceInput.LoadBalancers, updateInput.LoadBalancers)
		assert.Equal(t, serviceInput.ServiceRegistries, updateInput.ServiceRegistries)
		assert.Equal(t, serviceInput.VolumeConfigurations, updateInput.VolumeConfigurations)
		assert.Equal(t, serviceInput.PlacementConstraints, updateInput.PlacementConstraints)
		assert.Equal(t, serviceInput.PlacementStrategy, updateInput.PlacementStrategy)
		assert.Equal(t, serviceInput.VpcLatticeConfigurations, updateInput.VpcLatticeConfigurations)
	})
}
