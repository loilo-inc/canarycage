package rollout

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

func applyServiceDefinitionToUpdateInput(updateInput *ecs.UpdateServiceInput, serviceInput *ecs.CreateServiceInput) {
	// Preserve nil and empty slice distinctions from the service definition.
	updateInput.CapacityProviderStrategy = serviceInput.CapacityProviderStrategy
	updateInput.LoadBalancers = serviceInput.LoadBalancers
	updateInput.NetworkConfiguration = serviceInput.NetworkConfiguration
	updateInput.ServiceConnectConfiguration = serviceInput.ServiceConnectConfiguration
	updateInput.ServiceRegistries = serviceInput.ServiceRegistries
	updateInput.PlatformVersion = serviceInput.PlatformVersion
	updateInput.VolumeConfigurations = serviceInput.VolumeConfigurations
	updateInput.DeploymentConfiguration = serviceInput.DeploymentConfiguration
	updateInput.HealthCheckGracePeriodSeconds = serviceInput.HealthCheckGracePeriodSeconds
	updateInput.EnableECSManagedTags = &serviceInput.EnableECSManagedTags
	updateInput.PlacementConstraints = serviceInput.PlacementConstraints
	updateInput.PlacementStrategy = serviceInput.PlacementStrategy
	updateInput.PropagateTags = serviceInput.PropagateTags
	updateInput.VpcLatticeConfigurations = serviceInput.VpcLatticeConfigurations
}
