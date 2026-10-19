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

func toDockerPlatform(p *ecstypes.RuntimePlatform) (PlatformOS, PlatformArch) {
	var os PlatformOS = PlatformOSUnsupported
	var arch PlatformArch = PlatformArchUnsupported
	if p == nil ||
		p.OperatingSystemFamily == "" ||
		p.OperatingSystemFamily == ecstypes.OSFamilyLinux {
		os = PlatformOSLinux
	}
	if p == nil ||
		p.CpuArchitecture == "" ||
		p.CpuArchitecture == ecstypes.CPUArchitectureX8664 {
		arch = PlatformArchAmd64
	} else if p.CpuArchitecture == ecstypes.CPUArchitectureArm64 {
		arch = PlatformArchArm64
	}
	return os, arch
}
