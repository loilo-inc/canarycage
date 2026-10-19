package audit

import (
	"testing"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
)

func TestIsMediaTypeList(t *testing.T) {
	assert.True(t, isMediaTypeList("application/vnd.docker.distribution.manifest.list.v2+json"))
	assert.True(t, isMediaTypeList("application/vnd.oci.image.index.v1+json"))
	assert.False(t, isMediaTypeList("application/vnd.oci.image.index.v2+json"))
	assert.False(t, isMediaTypeList("application/vnd.docker.distribution.manifest.v2+json"))
}

func TestToDockerPlatform(t *testing.T) {
	for _, tc := range []struct {
		name     string
		platform *ecstypes.RuntimePlatform
		wantOS   PlatformOS
		wantArch PlatformArch
	}{
		{name: "nil defaults to Linux amd64", wantOS: PlatformOSLinux, wantArch: PlatformArchAmd64},
		{name: "empty defaults to Linux amd64", platform: &ecstypes.RuntimePlatform{}, wantOS: PlatformOSLinux, wantArch: PlatformArchAmd64},
		{name: "ARM64 with default OS", platform: &ecstypes.RuntimePlatform{CpuArchitecture: ecstypes.CPUArchitectureArm64}, wantOS: PlatformOSLinux, wantArch: PlatformArchArm64},
		{name: "Linux with default architecture", platform: &ecstypes.RuntimePlatform{OperatingSystemFamily: ecstypes.OSFamilyLinux}, wantOS: PlatformOSLinux, wantArch: PlatformArchAmd64},
		{name: "Linux X86_64", platform: &ecstypes.RuntimePlatform{OperatingSystemFamily: ecstypes.OSFamilyLinux, CpuArchitecture: ecstypes.CPUArchitectureX8664}, wantOS: PlatformOSLinux, wantArch: PlatformArchAmd64},
		{name: "Linux ARM64", platform: &ecstypes.RuntimePlatform{OperatingSystemFamily: ecstypes.OSFamilyLinux, CpuArchitecture: ecstypes.CPUArchitectureArm64}, wantOS: PlatformOSLinux, wantArch: PlatformArchArm64},
		{name: "Windows core", platform: &ecstypes.RuntimePlatform{OperatingSystemFamily: ecstypes.OSFamilyWindowsServer2022Core, CpuArchitecture: ecstypes.CPUArchitectureX8664}, wantOS: PlatformOSUnsupported, wantArch: PlatformArchAmd64},
		{name: "Windows full", platform: &ecstypes.RuntimePlatform{OperatingSystemFamily: ecstypes.OSFamilyWindowsServer2019Full}, wantOS: PlatformOSUnsupported, wantArch: PlatformArchAmd64},
		{name: "unknown OS preserves architecture", platform: &ecstypes.RuntimePlatform{OperatingSystemFamily: "UNKNOWN", CpuArchitecture: ecstypes.CPUArchitectureArm64}, wantOS: PlatformOSUnsupported, wantArch: PlatformArchArm64},
		{name: "unknown architecture preserves OS", platform: &ecstypes.RuntimePlatform{OperatingSystemFamily: ecstypes.OSFamilyLinux, CpuArchitecture: "RISCV64"}, wantOS: PlatformOSLinux, wantArch: PlatformArchUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os, arch := toDockerPlatform(tc.platform)
			assert.Equal(t, tc.wantOS, os)
			assert.Equal(t, tc.wantArch, arch)
		})
	}
}
