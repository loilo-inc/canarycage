package scan

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
)

const dockerManifestListMediaType = "application/vnd.docker.distribution.manifest.list.v2+json"

type ecrTool struct {
	Ecr awsiface.EcrClient
}

type EcrTool interface {
	GetActualImageIdentifier(ctx context.Context, info *ImageInfo) (*ecrtypes.ImageIdentifier, error)
	GetImageScanFindings(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier) (*ecrtypes.ImageScanFindings, error)
}

func newEcrTool(ecrClient awsiface.EcrClient) EcrTool {
	return &ecrTool{Ecr: ecrClient}
}

func (t *ecrTool) GetActualImageIdentifier(ctx context.Context, info *ImageInfo) (*ecrtypes.ImageIdentifier, error) {
	res, err := t.Ecr.BatchGetImage(ctx, &ecr.BatchGetImageInput{
		RepositoryName: aws.String(info.Repository),
		ImageIds:       []ecrtypes.ImageIdentifier{{ImageTag: aws.String(info.Tag)}},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Images) == 0 || res.Images[0].ImageManifest == nil {
		return nil, fmt.Errorf("image manifest not found for %s:%s", info.Repository, info.Tag)
	}

	var manifest dockerSchema
	if err := json.Unmarshal([]byte(*res.Images[0].ImageManifest), &manifest); err != nil {
		return nil, fmt.Errorf("parse image manifest for %s:%s: %w", info.Repository, info.Tag, err)
	}

	if manifest.MediaType == dockerManifestListMediaType {
		for _, candidate := range manifest.Manifests {
			if candidate.Platform == nil {
				continue
			}
			if toCPUArchitecture(candidate.Platform.Architecture) == info.PlatformArch {
				return &ecrtypes.ImageIdentifier{ImageDigest: aws.String(candidate.Digest)}, nil
			}
		}
		return nil, fmt.Errorf("no image found for architecture: %s in %s:%s", info.PlatformArch, info.Repository, info.Tag)
	}

	return &ecrtypes.ImageIdentifier{ImageTag: aws.String(info.Tag)}, nil
}

func (t *ecrTool) GetImageScanFindings(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier) (*ecrtypes.ImageScanFindings, error) {
	res, err := t.Ecr.DescribeImageScanFindings(ctx, &ecr.DescribeImageScanFindingsInput{
		RepositoryName: aws.String(info.Repository),
		ImageId:        imageID,
	})
	if err != nil {
		return nil, err
	}
	if res.ImageScanFindings == nil {
		return nil, fmt.Errorf("image scan findings missing for %s:%s", info.Repository, info.Tag)
	}
	return res.ImageScanFindings, nil
}

var _ awsiface.EcrClient = (*ecr.Client)(nil)

type dockerSchema struct {
	SchemaVersion int              `json:"schemaVersion"`
	MediaType     string           `json:"mediaType"`
	Manifests     []dockerManifest `json:"manifests,omitempty"`
	Config        *dockerManifest  `json:"config,omitempty"`
	Layers        []dockerManifest `json:"layers,omitempty"`
}

type dockerManifest struct {
	MediaType string          `json:"mediaType"`
	Size      int64           `json:"size"`
	Digest    string          `json:"digest"`
	Platform  *dockerPlatform `json:"platform,omitempty"`
}

type dockerPlatform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

func toCPUArchitecture(arch string) ecstypes.CPUArchitecture {
	switch arch {
	case "amd64":
		return ecstypes.CPUArchitectureX8664
	case "arm64":
		return ecstypes.CPUArchitectureArm64
	default:
		return ""
	}
}
