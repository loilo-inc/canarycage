package scan

import (
	"context"

	"github.com/apex/log"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/awsiface"
)

type scanner struct {
	ecs awsiface.EcsClient
	ecr awsiface.EcrClient
}

type Scanner interface {
	Scan(ctx context.Context, cluster string, service string) ([]*ScanResult, error)
}

type ScanResult struct {
	ImageInfo         *ImageInfo
	ImageScanFindings *ecrtypes.ImageScanFindings
	Err               error
}

func NewScanner(ecs awsiface.EcsClient, ecr awsiface.EcrClient) Scanner {
	return &scanner{ecs: ecs, ecr: ecr}
}

func (s *scanner) Scan(
	ctx context.Context,
	cluster string,
	service string,
) (results []*ScanResult, err error) {
	ecsTool := newEcsTool(s.ecs)
	ecrTool := newEcrTool(s.ecr)
	log.Infof("Scanning ECR image vulnerabilities for ECS service %s/%s", cluster, service)
	var imageInfos []*ImageInfo
	if imageInfos, err = ecsTool.GetServiceImageInfos(ctx, cluster, service); err != nil {
		return nil, err
	}
	findingsList := make([]*ScanResult, len(imageInfos))
	for i, info := range imageInfos {
		findingsList[i] = scanImage(ctx, ecrTool, info)
		findingsList[i].ImageInfo = imageInfos[i]
	}
	return findingsList, nil
}

func scanImage(ctx context.Context, ecrTool EcrTool, info *ImageInfo) *ScanResult {
	if imageID, err := ecrTool.GetActualImageIdentifier(ctx, info); err != nil {
		return &ScanResult{Err: err}
	} else if findings, err := ecrTool.GetImageScanFindings(ctx, info, imageID); err != nil {
		return &ScanResult{Err: err}
	} else {
		return &ScanResult{ImageScanFindings: findings}
	}
}
