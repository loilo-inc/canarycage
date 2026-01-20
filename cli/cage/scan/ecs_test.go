package scan

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/mocks/mock_awsiface"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetServiceImageInfos(t *testing.T) {
	ctx := context.Background()

	t.Run("returns image infos with default architecture", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcsClient(ctrl)
		tool := newEcsTool(mockClient)

		mockClient.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			DoAndReturn(func(ctx context.Context, input *ecs.DescribeServicesInput, opts ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
				assert.Equal(t, "cluster-a", *input.Cluster)
				assert.Equal(t, []string{"service-a"}, input.Services)
				return &ecs.DescribeServicesOutput{
					Services: []ecstypes.Service{{TaskDefinition: aws.String("td:1")}},
				}, nil
			})

		mockClient.EXPECT().DescribeTaskDefinition(ctx, gomock.AssignableToTypeOf(&ecs.DescribeTaskDefinitionInput{})).
			DoAndReturn(func(ctx context.Context, input *ecs.DescribeTaskDefinitionInput, opts ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error) {
				assert.Equal(t, "td:1", *input.TaskDefinition)
				return &ecs.DescribeTaskDefinitionOutput{
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
				}, nil
			})

		result, err := tool.GetServiceImageInfos(ctx, "cluster-a", "service-a")

		assert.NoError(t, err)
		if assert.Len(t, result, 2) {
			assert.Equal(t, "app", result[0].ContainerName)
			assert.Equal(t, ecstypes.CPUArchitectureX8664, result[0].PlatformArch)
			assert.Equal(t, "123456789012.dkr.ecr.us-west-2.amazonaws.com", result[0].Registry)
			assert.Equal(t, "my-repo", result[0].Repository)
			assert.Equal(t, "1.2.3", result[0].Tag)
			assert.Equal(t, "sidecar", result[1].ContainerName)
			assert.Equal(t, "", result[1].Registry)
			assert.Equal(t, "nginx", result[1].Repository)
			assert.Equal(t, "latest", result[1].Tag)
		}
	})

	t.Run("uses runtime platform architecture when provided", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcsClient(ctrl)
		tool := newEcsTool(mockClient)

		mockClient.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{{TaskDefinition: aws.String("td:2")}},
			}, nil)

		mockClient.EXPECT().DescribeTaskDefinition(ctx, gomock.AssignableToTypeOf(&ecs.DescribeTaskDefinitionInput{})).
			Return(&ecs.DescribeTaskDefinitionOutput{
				TaskDefinition: &ecstypes.TaskDefinition{
					RuntimePlatform: &ecstypes.RuntimePlatform{CpuArchitecture: ecstypes.CPUArchitectureArm64},
					ContainerDefinitions: []ecstypes.ContainerDefinition{
						{
							Name:  aws.String("app"),
							Image: aws.String("repo:v2"),
						},
					},
				},
			}, nil)

		result, err := tool.GetServiceImageInfos(ctx, "cluster-a", "service-a")

		assert.NoError(t, err)
		if assert.Len(t, result, 1) {
			assert.Equal(t, ecstypes.CPUArchitectureArm64, result[0].PlatformArch)
		}
	})

	t.Run("DescribeServices error returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcsClient(ctrl)
		tool := newEcsTool(mockClient)

		mockClient.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(nil, errors.New("API error"))

		result, err := tool.GetServiceImageInfos(ctx, "cluster-a", "service-a")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("service not found returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcsClient(ctrl)
		tool := newEcsTool(mockClient)

		mockClient.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(&ecs.DescribeServicesOutput{Services: []ecstypes.Service{}}, nil)

		result, err := tool.GetServiceImageInfos(ctx, "cluster-a", "service-a")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("missing task definition returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcsClient(ctrl)
		tool := newEcsTool(mockClient)

		mockClient.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{{TaskDefinition: nil}},
			}, nil)

		result, err := tool.GetServiceImageInfos(ctx, "cluster-a", "service-a")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("DescribeTaskDefinition error returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcsClient(ctrl)
		tool := newEcsTool(mockClient)

		mockClient.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{{TaskDefinition: aws.String("td:1")}},
			}, nil)

		mockClient.EXPECT().DescribeTaskDefinition(ctx, gomock.AssignableToTypeOf(&ecs.DescribeTaskDefinitionInput{})).
			Return(nil, errors.New("API error"))

		result, err := tool.GetServiceImageInfos(ctx, "cluster-a", "service-a")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("nil task definition returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcsClient(ctrl)
		tool := newEcsTool(mockClient)

		mockClient.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{{TaskDefinition: aws.String("td:1")}},
			}, nil)

		mockClient.EXPECT().DescribeTaskDefinition(ctx, gomock.AssignableToTypeOf(&ecs.DescribeTaskDefinitionInput{})).
			Return(&ecs.DescribeTaskDefinitionOutput{TaskDefinition: nil}, nil)

		result, err := tool.GetServiceImageInfos(ctx, "cluster-a", "service-a")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "task definition not found")
	})

	t.Run("no container definitions returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcsClient(ctrl)
		tool := newEcsTool(mockClient)

		mockClient.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{{TaskDefinition: aws.String("td:1")}},
			}, nil)

		mockClient.EXPECT().DescribeTaskDefinition(ctx, gomock.AssignableToTypeOf(&ecs.DescribeTaskDefinitionInput{})).
			Return(&ecs.DescribeTaskDefinitionOutput{
				TaskDefinition: &ecstypes.TaskDefinition{
					ContainerDefinitions: []ecstypes.ContainerDefinition{},
				},
			}, nil)

		result, err := tool.GetServiceImageInfos(ctx, "cluster-a", "service-a")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no container definitions")
	})

	t.Run("container definition missing name or image returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := mock_awsiface.NewMockEcsClient(ctrl)
		tool := newEcsTool(mockClient)

		mockClient.EXPECT().DescribeServices(ctx, gomock.AssignableToTypeOf(&ecs.DescribeServicesInput{})).
			Return(&ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{{TaskDefinition: aws.String("td:1")}},
			}, nil)

		mockClient.EXPECT().DescribeTaskDefinition(ctx, gomock.AssignableToTypeOf(&ecs.DescribeTaskDefinitionInput{})).
			Return(&ecs.DescribeTaskDefinitionOutput{
				TaskDefinition: &ecstypes.TaskDefinition{
					ContainerDefinitions: []ecstypes.ContainerDefinition{
						{
							Name:  nil,
							Image: aws.String("repo:v1"),
						},
					},
				},
			}, nil)

		result, err := tool.GetServiceImageInfos(ctx, "cluster-a", "service-a")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "container definition is missing name or image")
	})
}

func TestParseImageInfo(t *testing.T) {
	t.Run("parses registry, repository, and tag", func(t *testing.T) {
		parsed := ParseImageInfo("123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo:1.2.3")
		assert.Equal(t, "123456789012.dkr.ecr.us-west-2.amazonaws.com", parsed.Registry)
		assert.Equal(t, "my-repo", parsed.Repository)
		assert.Equal(t, "1.2.3", parsed.Tag)
	})

	t.Run("parses repository without registry", func(t *testing.T) {
		parsed := ParseImageInfo("nginx:latest")
		assert.Equal(t, "", parsed.Registry)
		assert.Equal(t, "nginx", parsed.Repository)
		assert.Equal(t, "latest", parsed.Tag)
	})

	t.Run("defaults tag to latest", func(t *testing.T) {
		parsed := ParseImageInfo("nginx")
		assert.Equal(t, "nginx", parsed.Repository)
		assert.Equal(t, "latest", parsed.Tag)
	})
}

func TestSplitRepoTag(t *testing.T) {
	t.Run("returns latest when tag is missing", func(t *testing.T) {
		repo, tag := splitRepoTag("repo")
		assert.Equal(t, "repo", repo)
		assert.Equal(t, "latest", tag)
	})

	t.Run("returns tag when present", func(t *testing.T) {
		repo, tag := splitRepoTag("repo:v1")
		assert.Equal(t, "repo", repo)
		assert.Equal(t, "v1", tag)
	})

	t.Run("returns latest when tag is empty", func(t *testing.T) {
		repo, tag := splitRepoTag("repo:")
		assert.Equal(t, "repo", repo)
		assert.Equal(t, "latest", tag)
	})
}
