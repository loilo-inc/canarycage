package rollout

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

func applyServiceDefinitionToUpdateInput(updateInput *ecs.UpdateServiceInput, serviceInput *ecs.CreateServiceInput) {
	// UpdateService treats omitted slice fields as "keep current", so explicitly
	// send empty slices for settings removed from service.json.
	updateInput.CapacityProviderStrategy = emptySliceIfNil(serviceInput.CapacityProviderStrategy)
	updateInput.LoadBalancers = emptySliceIfNil(serviceInput.LoadBalancers)
	updateInput.NetworkConfiguration = serviceInput.NetworkConfiguration
	updateInput.ServiceConnectConfiguration = serviceInput.ServiceConnectConfiguration
	updateInput.ServiceRegistries = emptySliceIfNil(serviceInput.ServiceRegistries)
	updateInput.PlatformVersion = serviceInput.PlatformVersion
	updateInput.VolumeConfigurations = emptySliceIfNil(serviceInput.VolumeConfigurations)
	updateInput.DeploymentConfiguration = serviceInput.DeploymentConfiguration
	updateInput.HealthCheckGracePeriodSeconds = serviceInput.HealthCheckGracePeriodSeconds
	updateInput.EnableECSManagedTags = &serviceInput.EnableECSManagedTags
	updateInput.PlacementConstraints = emptySliceIfNil(serviceInput.PlacementConstraints)
	updateInput.PlacementStrategy = emptySliceIfNil(serviceInput.PlacementStrategy)
	updateInput.PropagateTags = serviceInput.PropagateTags
	updateInput.VpcLatticeConfigurations = emptySliceIfNil(serviceInput.VpcLatticeConfigurations)
}

func emptySliceIfNil[T any](s []T) []T {
	if s != nil {
		return s
	}
	return []T{}
}
