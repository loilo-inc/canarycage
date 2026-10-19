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

func TestToCPUArchitecture(t *testing.T) {
	t.Run("amd64 maps to x86_64", func(t *testing.T) {
		assert.Equal(t, ecstypes.CPUArchitectureX8664, toCPUArchitecture("amd64"))
	})

	t.Run("arm64 maps to arm64", func(t *testing.T) {
		assert.Equal(t, ecstypes.CPUArchitectureArm64, toCPUArchitecture("arm64"))
	})

	t.Run("unknown architecture maps to empty", func(t *testing.T) {
		assert.Equal(t, ecstypes.CPUArchitecture(""), toCPUArchitecture("riscv"))
	})
}
