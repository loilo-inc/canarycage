package audit

import (
	"strings"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

const mediaTypeDockerList = "application/vnd.docker.distribution.manifest.list.v2+json"
const mediaTypeOCIList = "application/vnd.oci.image.index.v1+json"

func isMediaTypeList(mediaType string) bool {
	return mediaType == mediaTypeDockerList || mediaType == mediaTypeOCIList
}

type Schema struct {
	SchemaVersion int        `json:"schemaVersion"`
	MediaType     string     `json:"mediaType"`
	Manifests     []Manifest `json:"manifests,omitempty"`
	Config        *Manifest  `json:"config,omitempty"`
	Layers        []Manifest `json:"layers,omitempty"`
}

type Manifest struct {
	MediaType string    `json:"mediaType"`
	Size      int64     `json:"size"`
	Digest    string    `json:"digest"`
	Platform  *Platform `json:"platform,omitempty"`
	// only OCI manifests have annotations
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Platform struct {
	Architecture PlatformArch `json:"architecture"`
	OS           PlatformOS   `json:"os"`
	Variant      string       `json:"variant,omitempty"`
}

func toPlatformOS(os ecstypes.OSFamily) PlatformOS {
	if os == ecstypes.OSFamilyLinux {
		return PlatformOSLinux
	} else if strings.HasPrefix(string(os), "WINDOWS") {
		return PlatformOSWindows
	}
	return ""
}

func toPlatformArch(arch ecstypes.CPUArchitecture) PlatformArch {
	switch arch {
	case ecstypes.CPUArchitectureX8664:
		return PlatformArchAmd64
	case ecstypes.CPUArchitectureArm64:
		return PlatformArchArm64
	default:
		return ""
	}
}
