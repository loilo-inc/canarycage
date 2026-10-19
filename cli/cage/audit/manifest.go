package audit

import (
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
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
	Variant      string `json:"variant,omitempty"`
}

func toCPUArchitecture(arch string) ecstypes.CPUArchitecture {
	switch arch {
	case "amd64":
		return ecstypes.CPUArchitectureX8664
	case "arm64":
		return ecstypes.CPUArchitectureArm64
	default:
		return ""
	}
}
