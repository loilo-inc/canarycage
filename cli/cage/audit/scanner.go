package audit

import (
	"context"
	"fmt"

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
	var imageInfos []ImageInfo
	if imageInfos, err = ecsTool.GetServiceImageInfos(ctx, cluster, service); err != nil {
		return nil, err
	}
	findingsList := make([]*ScanResult, len(imageInfos))
	for i, info := range imageInfos {
		if info.IsECRImage() {
			findingsList[i] = scanImage(ctx, ecrTool, info)
		} else {
			findingsList[i] = &ScanResult{ImageInfo: info, Err: ErrNonEcrImage}
		}
	}
	return findingsList, nil
}

var ErrNonEcrImage = fmt.Errorf("non-ECR image")

func scanImage(ctx context.Context, ecrTool EcrTool, info ImageInfo) *ScanResult {
	if imageID, err := ecrTool.GetActualImageIdentifier(ctx, &info); err != nil {
		return &ScanResult{ImageInfo: info, Err: err}
	} else if findings, err := ecrTool.GetImageScanFindings(ctx, &info, imageID); err != nil {
		return &ScanResult{ImageInfo: info, Err: err}
	} else {
		return &ScanResult{ImageInfo: info, ImageScanFindings: findings}
	}
}
