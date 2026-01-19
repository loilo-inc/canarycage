package scan

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
)

type ecsTool struct {
	Ecs awsiface.EcsClient
}

type EcsTool interface {
	GetServiceImageInfos(ctx context.Context, cluster string, service string) ([]*ImageInfo, error)
}

func newEcsTool(ecsClient awsiface.EcsClient) EcsTool {
	return &ecsTool{Ecs: ecsClient}
}

func (t *ecsTool) GetServiceImageInfos(ctx context.Context, cluster string, service string) ([]*ImageInfo, error) {
	res, err := t.Ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &cluster,
		Services: []string{service},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Services) == 0 || res.Services[0].TaskDefinition == nil {
		return nil, fmt.Errorf("service not found: %s/%s", cluster, service)
	}
	taskDefinition := *res.Services[0].TaskDefinition

	tdRes, err := t.Ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: &taskDefinition,
	})
	if err != nil {
		return nil, err
	}
	td := tdRes.TaskDefinition
	if td == nil {
		return nil, fmt.Errorf("task definition not found: %s", taskDefinition)
	}

	arch := ecstypes.CPUArchitectureX8664
	if td.RuntimePlatform != nil && td.RuntimePlatform.CpuArchitecture != "" {
		arch = td.RuntimePlatform.CpuArchitecture
	}

	if len(td.ContainerDefinitions) == 0 {
		return nil, fmt.Errorf("no container definitions found for task definition: %s", taskDefinition)
	}

	images := make([]*ImageInfo, 0, len(td.ContainerDefinitions))
	for _, cd := range td.ContainerDefinitions {
		if cd.Name == nil || cd.Image == nil {
			return nil, fmt.Errorf("container definition is missing name or image: %s", taskDefinition)
		}
		parsed := ParseImageInfo(*cd.Image)
		images = append(images, &ImageInfo{
			ContainerName: *cd.Name,
			PlatformArch:  arch,
			Registry:      parsed.Registry,
			Repository:    parsed.Repository,
			Tag:           parsed.Tag,
		})
	}

	return images, nil
}

type ParsedImageInfo struct {
	Registry   string
	Repository string
	Tag        string
}

func ParseImageInfo(image string) ParsedImageInfo {
	parts := strings.Split(image, "/")
	if len(parts) == 1 {
		repository, tag := splitRepoTag(image)
		return ParsedImageInfo{Repository: repository, Tag: tag}
	}

	registry := parts[0]
	repoAndTag := strings.Join(parts[1:], "/")
	repository, tag := splitRepoTag(repoAndTag)
	return ParsedImageInfo{Registry: registry, Repository: repository, Tag: tag}
}

func splitRepoTag(value string) (string, string) {
	repository := value
	tag := "latest"
	if strings.Contains(value, ":") {
		parts := strings.SplitN(value, ":", 2)
		repository = parts[0]
		if parts[1] != "" {
			tag = parts[1]
		}
	}
	return repository, tag
}

var ecrURLPattern = regexp.MustCompile(`^\d{12}\.dkr\.ecr\.[a-z0-9-]+\.amazonaws\.com$`)

func IsEcr(registry string) bool {
	return ecrURLPattern.MatchString(registry)
}
