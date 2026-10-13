package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/v5/mocks/mock_awsiface"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetActualImageIdentifier(t *testing.T) {
	ctx := context.Background()

	t.Run("single architecture image returns tag", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcrClient(ctrl)
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		tool := newEcrTool(mockClient, nil)

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
		assert.Contains(t, err.Error(), "image scan findings missing")
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

type stubWaiter func(context.Context, *ImageInfo, time.Duration) (*ecr.DescribeImageScanFindingsOutput, error)

func (w stubWaiter) WaitForImageScan(ctx context.Context, info *ImageInfo, interval time.Duration) (*ecr.DescribeImageScanFindingsOutput, error) {
	return w(ctx, info, interval)
}

func TestExecImageScanFindings(t *testing.T) {
	findings := &ecrtypes.ImageScanFindings{Findings: []ecrtypes.ImageScanFinding{{Name: aws.String("CVE-TEST")}}}
	startErr := errors.New("start failed")
	waitErr := errors.New("wait failed")
	for _, tc := range []struct {
		name     string
		startErr error
		waitErr  error
		findings *ecrtypes.ImageScanFindings
		wantErr  string
	}{
		{name: "returns completed scan findings", findings: findings},
		{name: "returns start error without waiting", startErr: startErr, wantErr: startErr.Error()},
		{name: "returns waiter error", waitErr: waitErr, wantErr: waitErr.Error()},
		{name: "returns error for missing findings", wantErr: "image scan findings missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			info := &ImageInfo{Repository: "my-repo", Tag: "v1.0.0"}
			client := mock_awsiface.NewMockEcrClient(gomock.NewController(t))
			client.EXPECT().StartImageScan(ctx, &ecr.StartImageScanInput{
				RepositoryName: aws.String(info.Repository),
				ImageId:        &ecrtypes.ImageIdentifier{ImageTag: aws.String(info.Tag)},
			}).Return(&ecr.StartImageScanOutput{}, tc.startErr)
			waited := false
			tool := newEcrTool(client, stubWaiter(func(gotCtx context.Context, gotInfo *ImageInfo, interval time.Duration) (*ecr.DescribeImageScanFindingsOutput, error) {
				waited = true
				assert.Equal(t, ctx, gotCtx)
				assert.Equal(t, info, gotInfo)
				assert.Equal(t, 30*time.Second, interval)
				return &ecr.DescribeImageScanFindingsOutput{ImageScanFindings: tc.findings}, tc.waitErr
			}))
			got, err := tool.ExecImageScanFindings(ctx, info)
			assert.Equal(t, tc.startErr == nil, waited)
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Same(t, findings, got)
			}
		})
	}
}
