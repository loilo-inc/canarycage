package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type stubEcrTool struct {
	imageID  *ecrtypes.ImageIdentifier
	findings *ecrtypes.ImageScanFindings
	errID    error
	errScan  error
}

func (s *stubEcrTool) GetActualImageIdentifier(ctx context.Context, info *ImageInfo) (*ecrtypes.ImageIdentifier, error) {
	if s.errID != nil {
		return nil, s.errID
	}
	return s.imageID, nil
}

func (s *stubEcrTool) GetImageScanFindings(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier) (*ecrtypes.ImageScanFindings, error) {
	if s.errScan != nil {
		return nil, s.errScan
	}
	return s.findings, nil
}

func TestScanner_Scan(t *testing.T) {
	ctx := context.Background()

	t.Run("returns scan results for each image", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockEcs := mock_awsiface.NewMockEcsClient(ctrl)
		mockEcr := mock_awsiface.NewMockEcrClient(ctrl)
		scanner := NewScanner(mockEcs, mockEcr)

		mockEcs.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{{TaskDefinition: aws.String("td:1")}},
			}, nil)

		mockEcs.EXPECT().DescribeTaskDefinition(ctx, gomock.AssignableToTypeOf(&ecs.DescribeTaskDefinitionInput{})).
			Return(&ecs.DescribeTaskDefinitionOutput{
				TaskDefinition: &ecstypes.TaskDefinition{
					ContainerDefinitions: []ecstypes.ContainerDefinition{
						{
							Name:  aws.String("app"),
							Image: aws.String("123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo:1.2.3"),
						},
						{
							Name:  aws.String("sidecar"),
							Image: aws.String("nginx:latest"),
						},
					},
				},
			}, nil)

		manifestJSON, _ := json.Marshal(dockerSchema{
			SchemaVersion: 2,
			MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		})
		manifestStr := string(manifestJSON)

		mockEcr.EXPECT().BatchGetImage(ctx, gomock.AssignableToTypeOf(&ecr.BatchGetImageInput{})).
			DoAndReturn(func(ctx context.Context, input *ecr.BatchGetImageInput, opts ...func(*ecr.Options)) (*ecr.BatchGetImageOutput, error) {
				assert.Equal(t, "my-repo", *input.RepositoryName)
				assert.Equal(t, "1.2.3", *input.ImageIds[0].ImageTag)
				return &ecr.BatchGetImageOutput{
					Images: []ecrtypes.Image{{ImageManifest: &manifestStr}},
				}, nil
			})

		mockEcr.EXPECT().DescribeImageScanFindings(ctx, gomock.AssignableToTypeOf(&ecr.DescribeImageScanFindingsInput{})).
			DoAndReturn(func(ctx context.Context, input *ecr.DescribeImageScanFindingsInput, opts ...func(*ecr.Options)) (*ecr.DescribeImageScanFindingsOutput, error) {
				assert.Equal(t, "my-repo", *input.RepositoryName)
				assert.Equal(t, "1.2.3", *input.ImageId.ImageTag)
				return &ecr.DescribeImageScanFindingsOutput{
					ImageScanFindings: &ecrtypes.ImageScanFindings{},
				}, nil
			})

		results, err := scanner.Scan(ctx, "cluster-a", "service-a")

		assert.NoError(t, err)
		if assert.Len(t, results, 2) {
			assert.Equal(t, "app", results[0].ImageInfo.ContainerName)
			assert.NoError(t, results[0].Err)
			assert.Equal(t, "sidecar", results[1].ImageInfo.ContainerName)
			assert.EqualError(t, results[1].Err, "non-ECR image")
		}
	})

	t.Run("ecs error returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockEcs := mock_awsiface.NewMockEcsClient(ctrl)
		mockEcr := mock_awsiface.NewMockEcrClient(ctrl)
		scanner := NewScanner(mockEcs, mockEcr)

		mockEcs.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(nil, errors.New("ecs error"))

		results, err := scanner.Scan(ctx, "cluster-a", "service-a")

		assert.EqualError(t, err, "ecs error")
		assert.Nil(t, results)
	})
}

func TestScanImage(t *testing.T) {
	ctx := context.Background()

	t.Run("GetActualImageIdentifier error returns error", func(t *testing.T) {
		tool := &stubEcrTool{
			errID: errors.New("id error"),
		}

		result := scanImage(ctx, tool, &ImageInfo{Repository: "repo"})

		assert.EqualError(t, result.Err, "id error")
	})

	t.Run("GetImageScanFindings error returns error", func(t *testing.T) {
		tool := &stubEcrTool{
			imageID: &ecrtypes.ImageIdentifier{ImageTag: aws.String("v1")},
			errScan: errors.New("scan error"),
		}

		result := scanImage(ctx, tool, &ImageInfo{Repository: "repo"})

		assert.EqualError(t, result.Err, "scan error")
	})

	t.Run("success returns findings", func(t *testing.T) {
		findings := &ecrtypes.ImageScanFindings{}
		tool := &stubEcrTool{
			imageID:  &ecrtypes.ImageIdentifier{ImageTag: aws.String("v1")},
			findings: findings,
		}

		result := scanImage(ctx, tool, &ImageInfo{Repository: "repo"})

		assert.NoError(t, result.Err)
		assert.Equal(t, findings, result.ImageScanFindings)
	})
}
