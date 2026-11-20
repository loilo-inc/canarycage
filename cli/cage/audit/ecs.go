package audit

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/loilo-inc/canarycage/v5/awsiface"
)

type ecsTool struct {
	Ecs awsiface.EcsClient
}

type EcsTool interface {
	GetServiceImageInfos(ctx context.Context, cluster string, service string) ([]ImageInfo, error)
}

func newEcsTool(ecsClient awsiface.EcsClient) EcsTool {
	return &ecsTool{Ecs: ecsClient}
}

func (t *ecsTool) GetServiceImageInfos(ctx context.Context, cluster string, service string) ([]ImageInfo, error) {
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

	os, arch := toDockerPlatform(td.RuntimePlatform)

	if len(td.ContainerDefinitions) == 0 {
		return nil, fmt.Errorf("no container definitions found for task definition: %s", taskDefinition)
	}

	images := make([]ImageInfo, 0, len(td.ContainerDefinitions))
	for _, cd := range td.ContainerDefinitions {
		if cd.Name == nil || cd.Image == nil {
			return nil, fmt.Errorf("container definition is missing name or image: %s", taskDefinition)
		}
		parsed := ParseImageInfo(*cd.Image)
		images = append(images, ImageInfo{
			ContainerName: *cd.Name,
			PlatformOS:    os,
			PlatformArch:  arch,
			Registry:      parsed.Registry,
			Repository:    parsed.Repository,
			Tag:           parsed.Tag,
			Digest:        parsed.Digest,
		})
	}

	return images, nil
}

type ParsedImageInfo struct {
	Registry   string
	Repository string
	Tag        string
	Digest     string
}

func ParseImageInfo(image string) ParsedImageInfo {
	parts := strings.Split(image, "/")
	if len(parts) == 1 {
		parsedRef := splitRepoTag(image)
		return ParsedImageInfo{Repository: parsedRef.Repository, Tag: parsedRef.Tag, Digest: parsedRef.Digest}
	}

	registry := parts[0]
	var imageNoRegistry string
	if !strings.Contains(registry, ".") && !strings.Contains(registry, ":") && registry != "localhost" {
		registry = ""
		imageNoRegistry = strings.Join(parts, "/")
	} else {
		imageNoRegistry = strings.Join(parts[1:], "/")
	}
	parsedRef := splitRepoTag(imageNoRegistry)
	return ParsedImageInfo{Registry: registry, Repository: parsedRef.Repository, Tag: parsedRef.Tag, Digest: parsedRef.Digest}
}

func splitRepoTag(imageNoRegistry string) ParsedImageInfo {
	tag := "latest"
	ref := ParsedImageInfo{}
	value := imageNoRegistry
	// Check for digest first
	withDigest := strings.SplitN(value, "@", 2)
	if len(withDigest) == 2 {
		value = withDigest[0]
		ref.Digest = withDigest[1]
	}
	withTag := strings.SplitN(value, ":", 2)
	if len(withTag) == 2 {
		ref.Repository = withTag[0]
		if withTag[1] != "" {
			tag = withTag[1]
		}
	} else {
		ref.Repository = value
	}

	ref.Tag = tag
	return ref
}
