package scan

import (
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type ImageInfo struct {
	Registry      string
	ContainerName string
	PlatformArch  types.CPUArchitecture
	Repository    string
	Tag           string
}

type ScanResult struct {
	ImageInfo         *ImageInfo
	ImageScanFindings *ecrtypes.ImageScanFindings
	Err               error
}
