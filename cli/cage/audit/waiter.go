package audit

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/v5/awsiface"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/logger"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/loilo-inc/logos/v2/di"
)

type Waiter interface {
	WaitForImageScan(ctx context.Context, info *ImageInfo, interval time.Duration) (*ecr.DescribeImageScanFindingsOutput, error)
}

type waiter struct {
	di *di.D
}

var _ Waiter = (*waiter)(nil)

func (w *waiter) WaitForImageScan(ctx context.Context, info *ImageInfo, interval time.Duration) (*ecr.DescribeImageScanFindingsOutput, error) {
	time := w.di.Get(key.Time).(types.Time)
	ecrcli := w.di.Get(key.EcrCli).(awsiface.EcrClient)
	logger := w.di.Get(key.Logger).(logger.Logger)
	for {
		logger.Infof("Waiting for image scan: repository=%s, tag=%s", info.Repository, info.Tag)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.NewTimer(interval).C:
			res, err := ecrcli.DescribeImageScanFindings(ctx, &ecr.DescribeImageScanFindingsInput{
				RepositoryName: aws.String(info.Repository),
				ImageId: &ecrtypes.ImageIdentifier{
					ImageTag: aws.String(info.Tag),
				},
			})
			if err != nil {
				return nil, err
			}
			if res.ImageScanStatus != nil && res.ImageScanStatus.Status == ecrtypes.ScanStatusComplete {
				return res, nil
			}
		}
	}
}
