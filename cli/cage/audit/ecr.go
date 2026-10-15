package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/v5/awsiface"
)

const dockerManifestListMediaType = "application/vnd.docker.distribution.manifest.list.v2+json"

type ecrTool struct {
	Ecr    awsiface.EcrClient
	Waiter Waiter
}

type EcrTool interface {
	GetActualImageIdentifier(ctx context.Context, info *ImageInfo) (*ecrtypes.ImageIdentifier, error)
	GetImageScanFindings(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier) (*ecrtypes.ImageScanFindings, error)
	ExecImageScanFindings(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier) (*ecrtypes.ImageScanFindings, error)
}

func newEcrTool(ecrClient awsiface.EcrClient, scanWaiter Waiter) EcrTool {
	return &ecrTool{Ecr: ecrClient, Waiter: scanWaiter}
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
	return t.unwrapFindings(res)
}

func (t *ecrTool) ExecImageScanFindings(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier) (*ecrtypes.ImageScanFindings, error) {

	if res, err := t.prepareForImageScan(ctx, info, imageID); err != nil {
		if _, ok := errors.AsType[*ecrtypes.ScanNotFoundException](err); !ok {
			return nil, err
		}
	} else if !res.needsNewScan {
		return t.unwrapFindings(res.result)
	}

	if _, err := t.Ecr.StartImageScan(ctx, &ecr.StartImageScanInput{
		RepositoryName: aws.String(info.Repository),
		ImageId:        imageID,
	}); err != nil {
		return nil, err
	}

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if res, err := t.Waiter.WaitForImageScan(waitCtx, info, imageID, 30*time.Second); err != nil {
		return nil, err
	} else {
		return t.unwrapFindings(res)
	}
}

type prepareForImageScanResult struct {
	result       *ecr.DescribeImageScanFindingsOutput
	needsNewScan bool
}

func (t *ecrTool) prepareForImageScan(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier) (*prepareForImageScanResult, error) {
	res, err := t.Ecr.DescribeImageScanFindings(ctx, &ecr.DescribeImageScanFindingsInput{
		RepositoryName: aws.String(info.Repository),
		ImageId:        imageID,
	})

	if err != nil {
		return nil, err
	}

	if res.ImageScanStatus != nil &&
		(res.ImageScanStatus.Status == ecrtypes.ScanStatusInProgress ||
			res.ImageScanStatus.Status == ecrtypes.ScanStatusPending) {
		waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		if res, err = t.Waiter.WaitForImageScan(waitCtx, info, imageID, 30*time.Second); err != nil {
			return nil, err
		}
		return &prepareForImageScanResult{
			result:       res,
			needsNewScan: false,
		}, nil
	}

	outdated := true
	if res.ImageScanFindings != nil && res.ImageScanFindings.ImageScanCompletedAt != nil {
		completedAt := *res.ImageScanFindings.ImageScanCompletedAt
		outdated = time.Since(completedAt) > 24*time.Hour
	}

	return &prepareForImageScanResult{
		result:       res,
		needsNewScan: outdated,
	}, nil
}

func (t *ecrTool) unwrapFindings(res *ecr.DescribeImageScanFindingsOutput) (*ecrtypes.ImageScanFindings, error) {
	if res.ImageScanFindings == nil {
		return nil, fmt.Errorf("image scan findings missing")
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
