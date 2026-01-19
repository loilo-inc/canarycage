package scan

import "github.com/aws/aws-sdk-go-v2/service/ecs/types"

type ImageInfo struct {
	Registry      string
	ContainerName string
	PlatformArch  types.CPUArchitecture
	Repository    string
	Tag           string
}
