package audit

import (
	"context"
	"errors"
	"fmt"

	smithy "github.com/aws/smithy-go"
	"github.com/loilo-inc/canarycage/v5/awsiface"
)

type scanner struct {
	ecs awsiface.EcsClient
	ecr awsiface.EcrClient
}

type Scanner interface {
	Scan(ctx context.Context, cluster string, service string) ([]ScanResult, error)
}

var _ Scanner = (*scanner)(nil)

func NewScanner(ecs awsiface.EcsClient, ecr awsiface.EcrClient) Scanner {
	return &scanner{ecs: ecs, ecr: ecr}
}

func (s *scanner) Scan(
	ctx context.Context,
	cluster string,
	service string,
) (results []ScanResult, err error) {
	ecsTool := newEcsTool(s.ecs)
	ecrTool := newEcrTool(s.ecr)
	var imageInfos []ImageInfo
	if imageInfos, err = ecsTool.GetServiceImageInfos(ctx, cluster, service); err != nil {
		return nil, err
	}
	findingsList := make([]ScanResult, len(imageInfos))
	for i, info := range imageInfos {
		if info.IsECRImage() {
			findingsList[i] = scanImage(ctx, ecrTool, info)
		} else {
			findingsList[i] = ScanResult{ImageInfo: info, Err: ErrNonEcrImage}
		}
	}
	return findingsList, nil
}

var ErrNonEcrImage = fmt.Errorf("non-ECR image")
var ErrScanNotFound = fmt.Errorf("scan not found")

func scanImage(ctx context.Context, ecrTool EcrTool, info ImageInfo) ScanResult {
	if imageID, err := ecrTool.GetActualImageIdentifier(ctx, &info); err != nil {
		return ScanResult{ImageInfo: info, Err: err}
	} else if findings, err := ecrTool.GetImageScanFindings(ctx, &info, imageID); err != nil {
		return ScanResult{ImageInfo: info, Err: parseError(err)}
	} else {
		var cves []CVE
		for _, f := range findings.Findings {
			cve := findingToCVE(f)
			cves = append(cves, cve)
		}
		return ScanResult{ImageInfo: info, Cves: cves}
	}
}

func parseError(err error) error {
	var awserr smithy.APIError
	if errors.As(err, &awserr) && awserr.ErrorCode() == "ScanNotFoundException" {
		return ErrScanNotFound
	}
	return err
}
