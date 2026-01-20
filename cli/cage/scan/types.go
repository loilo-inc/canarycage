package scan

import (
	"regexp"

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

func (i *ImageInfo) IsECRImage() bool {
	return i.Registry == "public.ecr.aws" || i.registryHasECRSuffix()
}

func (i *ImageInfo) registryHasECRSuffix() bool {
	pat := regexp.MustCompile(`^[0-9]{12}\.dkr\.ecr\.[a-z0-9-]+\.amazonaws\.com$`)
	return pat.MatchString(i.Registry)
}

type ScanResult struct {
	ImageInfo         *ImageInfo
	ImageScanFindings *ecrtypes.ImageScanFindings
	Err               error
}
