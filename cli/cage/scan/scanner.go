package scan

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/loilo-inc/canarycage/awsiface"
)

type scanner struct {
	ecs awsiface.EcsClient
	ecr awsiface.EcrClient
}

type Scanner interface {
	Scan(ctx context.Context, cluster string, service string) ([]*ScanResult, error)
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
		if info.IsECRImage() {
			findingsList[i] = scanImage(ctx, ecrTool, info)
			findingsList[i].ImageInfo = imageInfos[i]
		} else {
			findingsList[i] = &ScanResult{ImageInfo: info, Err: fmt.Errorf("non-ECR image")}
		}
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
