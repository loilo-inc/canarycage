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

func TestToPlatformOS(t *testing.T) {
	for _, tc := range []struct {
		name string
		os   ecstypes.OSFamily
		want PlatformOS
	}{
		{name: "Linux", os: ecstypes.OSFamilyLinux, want: PlatformOSLinux},
		{name: "Windows core", os: ecstypes.OSFamilyWindowsServer2022Core, want: PlatformOSWindows},
		{name: "Windows full", os: ecstypes.OSFamilyWindowsServer2019Full, want: PlatformOSWindows},
		{name: "empty", os: "", want: ""},
		{name: "unknown", os: "UNKNOWN", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, toPlatformOS(tc.os))
		})
	}
}

func TestToPlatformArch(t *testing.T) {
	for _, tc := range []struct {
		name string
		arch ecstypes.CPUArchitecture
		want PlatformArch
	}{
		{name: "X86_64", arch: ecstypes.CPUArchitectureX8664, want: PlatformArchAmd64},
		{name: "ARM64", arch: ecstypes.CPUArchitectureArm64, want: PlatformArchArm64},
		{name: "empty", arch: "", want: ""},
		{name: "unknown", arch: "RISCV64", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, toPlatformArch(tc.arch))
		})
	}
}
