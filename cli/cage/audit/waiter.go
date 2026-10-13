package audit

import (
	"context"
	"errors"
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
	WaitForImageScan(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier, interval time.Duration) (*ecr.DescribeImageScanFindingsOutput, error)
}

type waiter struct {
	di *di.D
}

var _ Waiter = (*waiter)(nil)

func (w *waiter) WaitForImageScan(ctx context.Context, info *ImageInfo, imageID *ecrtypes.ImageIdentifier, interval time.Duration) (*ecr.DescribeImageScanFindingsOutput, error) {
	time := w.di.Get(key.Time).(types.Time)
	ecrcli := w.di.Get(key.EcrCli).(awsiface.EcrClient)
	logger := w.di.Get(key.Logger).(logger.Logger)
	for {
		logger.Errorf("Waiting for image scan: repository=%s, tag=%s", info.Repository, info.Tag)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.NewTimer(interval).C:
			res, err := ecrcli.DescribeImageScanFindings(ctx, &ecr.DescribeImageScanFindingsInput{
				RepositoryName: aws.String(info.Repository),
				ImageId:        imageID,
			})
			if err != nil {
				return nil, err
			} else if res.ImageScanStatus == nil {
				return nil, errors.New("image scan status missing")
			}

			status := res.ImageScanStatus.Status

			switch status {
			case ecrtypes.ScanStatusInProgress:
				continue
			case ecrtypes.ScanStatusComplete:
				return res, nil
			default:
				return nil, errors.New("image scan failed: status=" + string(status))
			}
		}
	}
}
