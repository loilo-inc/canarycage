package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type stubWaiter func(context.Context, *ImageInfo, *ecrtypes.ImageIdentifier, time.Duration) (*ecr.DescribeImageScanFindingsOutput, error)

func (w stubWaiter) WaitForImageScan(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier, interval time.Duration) (*ecr.DescribeImageScanFindingsOutput, error) {
	return w(ctx, info, imageID, interval)
}

func TestExecImageScanFindings(t *testing.T) {
	freshAt := time.Now().Add(-time.Hour)
	oldAt := time.Now().Add(-25 * time.Hour)
	findings := &ecrtypes.ImageScanFindings{Findings: []ecrtypes.ImageScanFinding{{Name: aws.String("CVE-TEST")}}}
	fresh := &ecr.DescribeImageScanFindingsOutput{
		ImageScanStatus:   &ecrtypes.ImageScanStatus{Status: ecrtypes.ScanStatusComplete},
		ImageScanFindings: &ecrtypes.ImageScanFindings{ImageScanCompletedAt: &freshAt, Findings: findings.Findings},
	}
	old := &ecr.DescribeImageScanFindingsOutput{
		ImageScanStatus:   &ecrtypes.ImageScanStatus{Status: ecrtypes.ScanStatusComplete},
		ImageScanFindings: &ecrtypes.ImageScanFindings{ImageScanCompletedAt: &oldAt},
	}
	startErr := errors.New("start failed")
	waitErr := errors.New("wait failed")
	describeErr := errors.New("access denied")
	notFound := fmt.Errorf("describe: %w", &ecrtypes.ScanNotFoundException{})
	for _, tc := range []struct {
		name         string
		initial      *ecr.DescribeImageScanFindingsOutput
		describeErr  error
		start        bool
		wait         bool
		startErr     error
		waitErr      error
		wantFindings *ecrtypes.ImageScanFindings
		wantErr      string
	}{
		{name: "reuses scan completed within 24 hours", initial: fresh, wantFindings: fresh.ImageScanFindings},
		{name: "rescans results older than 24 hours", initial: old, start: true, wait: true, wantFindings: findings},
		{name: "starts first scan when results are not found", describeErr: notFound, start: true, wait: true, wantFindings: findings},
		{name: "starts scan when completed timestamp is missing", initial: &ecr.DescribeImageScanFindingsOutput{ImageScanFindings: findings}, start: true, wait: true, wantFindings: findings},
		{name: "starts scan when findings are missing", initial: &ecr.DescribeImageScanFindingsOutput{}, start: true, wait: true, wantFindings: findings},
		{name: "waits for first scan already in progress", initial: &ecr.DescribeImageScanFindingsOutput{ImageScanStatus: &ecrtypes.ImageScanStatus{Status: ecrtypes.ScanStatusInProgress}}, wait: true, wantFindings: findings},
		{name: "propagates describe error without starting", describeErr: describeErr, wantErr: describeErr.Error()},
		{name: "returns start error without waiting", initial: old, start: true, startErr: startErr, wantErr: startErr.Error()},
		{name: "returns waiter error after starting", initial: old, start: true, wait: true, waitErr: waitErr, wantErr: waitErr.Error()},
		{name: "returns waiter error for existing scan", initial: &ecr.DescribeImageScanFindingsOutput{ImageScanStatus: &ecrtypes.ImageScanStatus{Status: ecrtypes.ScanStatusInProgress}}, wait: true, waitErr: waitErr, wantErr: waitErr.Error()},
		{name: "returns error for missing findings after scan", initial: old, start: true, wait: true, wantErr: "image scan findings missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			info := &ImageInfo{Repository: "my-repo", Tag: "v1.0.0"}
			imageID := &ecrtypes.ImageIdentifier{ImageDigest: aws.String("sha256:arm64digest")}
			client := mock_awsiface.NewMockEcrClient(gomock.NewController(t))
			describeCall := client.EXPECT().DescribeImageScanFindings(ctx, &ecr.DescribeImageScanFindingsInput{
				RepositoryName: aws.String(info.Repository), ImageId: imageID,
			}).Return(tc.initial, tc.describeErr)
			if tc.start {
				client.EXPECT().StartImageScan(ctx, &ecr.StartImageScanInput{
					RepositoryName: aws.String(info.Repository), ImageId: imageID,
				}).After(describeCall).Return(&ecr.StartImageScanOutput{}, tc.startErr)
			}
			waited := false
			var waitCtx context.Context
			before := time.Now()
			tool := newEcrTool(client, stubWaiter(func(gotCtx context.Context, gotInfo *ImageInfo, gotImageID *ecrtypes.ImageIdentifier, interval time.Duration) (*ecr.DescribeImageScanFindingsOutput, error) {
				waited = true
				waitCtx = gotCtx
				deadline, ok := gotCtx.Deadline()
				assert.True(t, ok)
				assert.WithinDuration(t, before.Add(5*time.Minute), deadline, time.Second)
				assert.NoError(t, gotCtx.Err())
				assert.Same(t, imageID, gotImageID)
				assert.Equal(t, info, gotInfo)
				assert.Equal(t, 30*time.Second, interval)
				return &ecr.DescribeImageScanFindingsOutput{ImageScanFindings: tc.wantFindings}, tc.waitErr
			}))
			got, err := tool.ExecImageScanFindings(ctx, info, imageID)
			assert.Equal(t, tc.wait, waited)
			if waited {
				assert.ErrorIs(t, waitCtx.Err(), context.Canceled)
			}
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Same(t, tc.wantFindings, got)
			}
		})
	}
}

func TestGetImageScanFindings_NotFound(t *testing.T) {
	client := mock_awsiface.NewMockEcrClient(gomock.NewController(t))
	info := &ImageInfo{Repository: "my-repo", Tag: "v1.0.0"}
	imageID := &ecrtypes.ImageIdentifier{ImageTag: aws.String(info.Tag)}
	expected := fmt.Errorf("describe: %w", &ecrtypes.ScanNotFoundException{})
	client.EXPECT().DescribeImageScanFindings(t.Context(), &ecr.DescribeImageScanFindingsInput{
		RepositoryName: aws.String(info.Repository), ImageId: imageID,
	}).Return(nil, expected)
	tool := newEcrTool(client, nil)
	got, err := tool.GetImageScanFindings(t.Context(), info, imageID)
	assert.ErrorIs(t, err, expected)
	assert.Nil(t, got)
}
