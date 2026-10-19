package audit

import (
	"context"
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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetActualImageIdentifier(t *testing.T) {
	const arm64Digest = "sha256:8c31ff948acc1d452447c7562927ad269241bb7f104510d9d5a6a28ea4a1c755"
	// Keep the relevant OCI fields from the reported Redis index, including
	// attestation entries and an ARM variant that are absent from Docker fixtures.
	const manifests = `"manifests":[
		{"digest":"sha256:noplatform"},
		{"digest":"sha256:attestation","platform":{"architecture":"unknown","os":"unknown"},
		 "annotations":{"vnd.docker.reference.type":"attestation-manifest"}},
		{"digest":"sha256:amd64digest","platform":{"architecture":"amd64","os":"linux"}},
		{"digest":"` + arm64Digest + `","platform":{"architecture":"arm64","os":"linux","variant":"v8"}}
	]`

	for _, mediaType := range []string{mediaTypeDockerList, mediaTypeOCIList} {
		t.Run(mediaType, func(t *testing.T) {
			for _, tc := range []struct {
				name       string
				arch       ecstypes.CPUArchitecture
				manifest   string
				wantDigest string
				wantErr    string
			}{
				{name: "ARM64 digest skips missing platform and attestation", arch: ecstypes.CPUArchitectureArm64,
					manifest: `{"schemaVersion":2,"mediaType":"` + mediaType + `",` + manifests + `}`, wantDigest: arm64Digest},
				{name: "AMD64 digest", arch: ecstypes.CPUArchitectureX8664,
					manifest: `{"schemaVersion":2,"mediaType":"` + mediaType + `",` + manifests + `}`, wantDigest: "sha256:amd64digest"},
				{name: "no matching architecture", arch: ecstypes.CPUArchitectureArm64,
					manifest: `{"schemaVersion":2,"manifests":[{"digest":"sha256:amd64digest","platform":{"architecture":"amd64","os":"linux"}}]}`,
					wantErr:  "no image found for architecture: ARM64 in my-repo:v1.0.0"},
				{name: "empty list", arch: ecstypes.CPUArchitectureArm64, manifest: `{"schemaVersion":2,"manifests":[]}`,
					wantErr: "no image found for architecture: ARM64 in my-repo:v1.0.0"},
				{name: "invalid JSON", manifest: "invalid json", wantErr: "parse image manifest for my-repo:v1.0.0"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					ctx := t.Context()
					client := mock_awsiface.NewMockEcrClient(gomock.NewController(t))
					info := &ImageInfo{Repository: "my-repo", Tag: "v1.0.0", PlatformArch: tc.arch}
					client.EXPECT().BatchGetImage(ctx, &ecr.BatchGetImageInput{
						RepositoryName: aws.String(info.Repository),
						ImageIds:       []ecrtypes.ImageIdentifier{{ImageTag: aws.String(info.Tag)}},
					}).Return(&ecr.BatchGetImageOutput{Images: []ecrtypes.Image{{
						ImageManifest: aws.String(tc.manifest), ImageManifestMediaType: aws.String(mediaType),
					}}}, nil)

					got, err := newEcrTool(client, nil).GetActualImageIdentifier(ctx, info)
					if tc.wantErr != "" {
						require.ErrorContains(t, err, tc.wantErr)
						assert.Nil(t, got)
						return
					}
					require.NoError(t, err)
					assert.Equal(t, &ecrtypes.ImageIdentifier{ImageDigest: aws.String(tc.wantDigest)}, got)
				})
			}
		})
	}

	for _, tc := range []struct {
		name    string
		images  []ecrtypes.Image
		apiErr  error
		wantErr string
	}{
		{name: "Docker single image", images: []ecrtypes.Image{{ImageManifest: aws.String(`{"schemaVersion":2}`), ImageManifestMediaType: aws.String(mediaTypeDockerTag)}}},
		{name: "OCI single image", images: []ecrtypes.Image{{ImageManifest: aws.String(`{"schemaVersion":2}`), ImageManifestMediaType: aws.String(mediaTypeOCITag)}}},
		{name: "API error", apiErr: errors.New("API error"), wantErr: "API error"},
		{name: "empty images", wantErr: "image manifest not found"},
		{name: "nil manifest", images: []ecrtypes.Image{{ImageManifestMediaType: aws.String(mediaTypeOCIList)}}, wantErr: "image manifest not found"},
		{name: "unsupported media type", images: []ecrtypes.Image{{ImageManifest: aws.String(`{}`), ImageManifestMediaType: aws.String("application/unknown")}}, wantErr: "unsupported media type: application/unknown"},
		{name: "nil media type", images: []ecrtypes.Image{{ImageManifest: aws.String(`{}`)}}, wantErr: "unsupported media type:"},
		{name: "empty media type", images: []ecrtypes.Image{{ImageManifest: aws.String(`{}`), ImageManifestMediaType: aws.String("")}}, wantErr: "unsupported media type:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			client := mock_awsiface.NewMockEcrClient(gomock.NewController(t))
			client.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).Return(&ecr.BatchGetImageOutput{Images: tc.images}, tc.apiErr)
			info := &ImageInfo{Repository: "my-repo", Tag: "v1.0.0", PlatformArch: ecstypes.CPUArchitectureArm64}
			got, err := newEcrTool(client, nil).GetActualImageIdentifier(ctx, info)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, &ecrtypes.ImageIdentifier{ImageTag: aws.String(info.Tag)}, got)
		})
	}
}

func TestGetImageScanFindings(t *testing.T) {
	ctx := context.Background()

	for _, status := range []ecrtypes.ScanStatus{ecrtypes.ScanStatusInProgress, ecrtypes.ScanStatusPending} {
		t.Run("does not wait for "+string(status), func(t *testing.T) {
			client := mock_awsiface.NewMockEcrClient(gomock.NewController(t))
			info := &ImageInfo{Repository: "my-repo", Tag: "v1.0.0"}
			imageID := &ecrtypes.ImageIdentifier{ImageTag: aws.String(info.Tag)}
			findings := &ecrtypes.ImageScanFindings{}
			client.EXPECT().DescribeImageScanFindings(ctx, &ecr.DescribeImageScanFindingsInput{
				RepositoryName: aws.String(info.Repository), ImageId: imageID,
			}).Return(&ecr.DescribeImageScanFindingsOutput{
				ImageScanStatus:   &ecrtypes.ImageScanStatus{Status: status},
				ImageScanFindings: findings,
			}, nil).Times(1)
			tool := newEcrTool(client, stubWaiter(func(context.Context, *ImageInfo, *ecrtypes.ImageIdentifier, time.Duration) (*ecr.DescribeImageScanFindingsOutput, error) {
				t.Fatal("ordinary audit must not wait for a scan")
				return nil, nil
			}))
			got, err := tool.GetImageScanFindings(ctx, info, imageID)
			assert.NoError(t, err)
			assert.Same(t, findings, got)
		})
	}

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
		{name: "waits for pending scan without starting another", initial: &ecr.DescribeImageScanFindingsOutput{ImageScanStatus: &ecrtypes.ImageScanStatus{Status: ecrtypes.ScanStatusPending}}, wait: true, wantFindings: findings},
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
