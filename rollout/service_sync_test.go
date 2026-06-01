package rollout

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
)

func TestApplyServiceDefinitionToUpdateInput(t *testing.T) {
	updateInput := &ecs.UpdateServiceInput{}
	applyServiceDefinitionToUpdateInput(updateInput, &ecs.CreateServiceInput{})

	assert.Equal(t, []ecstypes.CapacityProviderStrategyItem{}, updateInput.CapacityProviderStrategy)
	assert.Equal(t, []ecstypes.LoadBalancer{}, updateInput.LoadBalancers)
	assert.Equal(t, []ecstypes.ServiceRegistry{}, updateInput.ServiceRegistries)
	assert.Equal(t, []ecstypes.ServiceVolumeConfiguration{}, updateInput.VolumeConfigurations)
	assert.Equal(t, []ecstypes.PlacementConstraint{}, updateInput.PlacementConstraints)
	assert.Equal(t, []ecstypes.PlacementStrategy{}, updateInput.PlacementStrategy)
	assert.Equal(t, []ecstypes.VpcLatticeConfiguration{}, updateInput.VpcLatticeConfigurations)

	assert.Nil(t, updateInput.NetworkConfiguration)
	assert.Nil(t, updateInput.ServiceConnectConfiguration)
	assert.Nil(t, updateInput.PlatformVersion)
	assert.Nil(t, updateInput.DeploymentConfiguration)
	assert.Nil(t, updateInput.HealthCheckGracePeriodSeconds)
	assert.Equal(t, ecstypes.PropagateTags(""), updateInput.PropagateTags)
}
