package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetActualImageIdentifier(t *testing.T) {
	ctx := context.Background()

	t.Run("single architecture image returns tag", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository:   "my-repo",
			Tag:          "v1.0.0",
			PlatformArch: ecstypes.CPUArchitectureX8664,
		}

		manifest := dockerSchema{
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		}
		manifestJSON, _ := json.Marshal(manifest)
		manifestStr := string(manifestJSON)

		mockClient.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).Return(&ecr.BatchGetImageOutput{
			Images: []ecrtypes.Image{
				{ImageManifest: &manifestStr},
			},
		}, nil)

		result, err := tool.GetActualImageIdentifier(ctx, info)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "v1.0.0", *result.ImageTag)
	})

	t.Run("multi-arch image returns digest for matching architecture", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository:   "my-repo",
			Tag:          "v1.0.0",
			PlatformArch: ecstypes.CPUArchitectureArm64,
		}

		manifest := dockerSchema{
			SchemaVersion: 2,
			MediaType:     dockerManifestListMediaType,
			Manifests: []dockerManifest{
				{
					Digest: "sha256:amd64digest",
					Platform: &dockerPlatform{
						Architecture: "amd64",
						OS:           "linux",
					},
				},
				{
					Digest: "sha256:arm64digest",
					Platform: &dockerPlatform{
						Architecture: "arm64",
						OS:           "linux",
					},
				},
			},
		}
		manifestJSON, _ := json.Marshal(manifest)
		manifestStr := string(manifestJSON)

		mockClient.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).Return(&ecr.BatchGetImageOutput{
			Images: []ecrtypes.Image{
				{ImageManifest: &manifestStr},
			},
		}, nil)

		result, err := tool.GetActualImageIdentifier(ctx, info)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "sha256:arm64digest", *result.ImageDigest)
	})

	t.Run("multi-arch image with no matching architecture returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository:   "my-repo",
			Tag:          "v1.0.0",
			PlatformArch: ecstypes.CPUArchitectureArm64,
		}

		manifest := dockerSchema{
			SchemaVersion: 2,
			MediaType:     dockerManifestListMediaType,
			Manifests: []dockerManifest{
				{
					Digest: "sha256:amd64digest",
					Platform: &dockerPlatform{
						Architecture: "amd64",
						OS:           "linux",
					},
				},
			},
		}
		manifestJSON, _ := json.Marshal(manifest)
		manifestStr := string(manifestJSON)

		mockClient.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).Return(&ecr.BatchGetImageOutput{
			Images: []ecrtypes.Image{
				{ImageManifest: &manifestStr},
			},
		}, nil)

		result, err := tool.GetActualImageIdentifier(ctx, info)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no image found for architecture")
	})

	t.Run("BatchGetImage error returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository:   "my-repo",
			Tag:          "v1.0.0",
			PlatformArch: ecstypes.CPUArchitectureX8664,
		}

		mockClient.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).Return(nil, errors.New("API error"))

		result, err := tool.GetActualImageIdentifier(ctx, info)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("empty images returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository:   "my-repo",
			Tag:          "v1.0.0",
			PlatformArch: ecstypes.CPUArchitectureX8664,
		}

		mockClient.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).Return(&ecr.BatchGetImageOutput{
			Images: []ecrtypes.Image{},
		}, nil)

		result, err := tool.GetActualImageIdentifier(ctx, info)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "image manifest not found")
	})

	t.Run("nil manifest returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository:   "my-repo",
			Tag:          "v1.0.0",
			PlatformArch: ecstypes.CPUArchitectureX8664,
		}

		mockClient.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).Return(&ecr.BatchGetImageOutput{
			Images: []ecrtypes.Image{
				{ImageManifest: nil},
			},
		}, nil)

		result, err := tool.GetActualImageIdentifier(ctx, info)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "image manifest not found")
	})

	t.Run("invalid JSON manifest returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository:   "my-repo",
			Tag:          "v1.0.0",
			PlatformArch: ecstypes.CPUArchitectureX8664,
		}

		invalidJSON := "invalid json"
		mockClient.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).Return(&ecr.BatchGetImageOutput{
			Images: []ecrtypes.Image{
				{ImageManifest: &invalidJSON},
			},
		}, nil)

		result, err := tool.GetActualImageIdentifier(ctx, info)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "parse image manifest")
	})

	t.Run("multi-arch image skips manifests without platform", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository:   "my-repo",
			Tag:          "v1.0.0",
			PlatformArch: ecstypes.CPUArchitectureArm64,
		}

		manifest := dockerSchema{
			SchemaVersion: 2,
			MediaType:     dockerManifestListMediaType,
			Manifests: []dockerManifest{
				{
					Digest:   "sha256:noplatform",
					Platform: nil,
				},
				{
					Digest: "sha256:arm64digest",
					Platform: &dockerPlatform{
						Architecture: "arm64",
						OS:           "linux",
					},
				},
			},
		}
		manifestJSON, _ := json.Marshal(manifest)
		manifestStr := string(manifestJSON)

		mockClient.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).Return(&ecr.BatchGetImageOutput{
			Images: []ecrtypes.Image{
				{ImageManifest: &manifestStr},
			},
		}, nil)

		result, err := tool.GetActualImageIdentifier(ctx, info)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "sha256:arm64digest", *result.ImageDigest)
	})
}
func TestGetImageScanFindings(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully returns scan findings", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository: "my-repo",
			Tag:        "v1.0.0",
		}
		imageID := &ecrtypes.ImageIdentifier{
			ImageDigest: aws.String("sha256:abc123"),
		}

		expectedFindings := &ecrtypes.ImageScanFindings{
			FindingSeverityCounts: map[string]int32{
				"CRITICAL": 1,
				"HIGH":     2,
			},
		}

		mockClient.EXPECT().DescribeImageScanFindings(
			ctx, gomock.AssignableToTypeOf(&ecr.DescribeImageScanFindingsInput{})).
			DoAndReturn(func(ctx context.Context,
				input *ecr.DescribeImageScanFindingsInput,
				opts ...func(*ecr.Options)) (*ecr.DescribeImageScanFindingsOutput, error) {
				assert.Equal(t, "my-repo", *input.RepositoryName)
				assert.Equal(t, "sha256:abc123", *input.ImageId.ImageDigest)
				return &ecr.DescribeImageScanFindingsOutput{ImageScanFindings: expectedFindings}, nil
			})

		result, err := tool.GetImageScanFindings(ctx, info, imageID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedFindings, result)
	})

	t.Run("DescribeImageScanFindings error returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository: "my-repo",
			Tag:        "v1.0.0",
		}
		imageID := &ecrtypes.ImageIdentifier{
			ImageTag: aws.String("v1.0.0"),
		}

		mockClient.EXPECT().DescribeImageScanFindings(ctx, gomock.AssignableToTypeOf(&ecr.DescribeImageScanFindingsInput{})).Return(nil, errors.New("API error"))

		result, err := tool.GetImageScanFindings(ctx, info, imageID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("nil ImageScanFindings returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient)

		info := &ImageInfo{
			Repository: "my-repo",
			Tag:        "v1.0.0",
		}
		imageID := &ecrtypes.ImageIdentifier{
			ImageTag: aws.String("v1.0.0"),
		}

		mockClient.EXPECT().DescribeImageScanFindings(ctx, gomock.AssignableToTypeOf(&ecr.DescribeImageScanFindingsInput{})).Return(&ecr.DescribeImageScanFindingsOutput{
			ImageScanFindings: nil,
		}, nil)

		result, err := tool.GetImageScanFindings(ctx, info, imageID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "image scan findings missing for my-repo:v1.0.0")
	})
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
