package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/mocks/mock_awsiface"
	"github.com/loilo-inc/canarycage/v5/test"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/loilo-inc/logos/v2/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestWaitForImageScan(t *testing.T) {
	info := &ImageInfo{Repository: "my-repo", Tag: "v1.0.0"}
	imageID := &ecrtypes.ImageIdentifier{ImageDigest: aws.String("sha256:arm64digest")}
	input := &ecr.DescribeImageScanFindingsInput{
		RepositoryName: aws.String(info.Repository),
		ImageId:        imageID,
	}
	setup := func(t *testing.T, clock types.Time) (*waiter, *mock_awsiface.MockEcrClient) {
		t.Helper()
		client := mock_awsiface.NewMockEcrClient(gomock.NewController(t))
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.EcrCli, client)
			b.Set(key.Time, clock)
			b.Set(key.Logger, test.NewLogger())
		})
		return &waiter{di: d}, client
	}
	t.Run("polls until scan completes", func(t *testing.T) {
		w, client := setup(t, test.NewFakeTime())
		completed := &ecr.DescribeImageScanFindingsOutput{
			ImageScanStatus:   &ecrtypes.ImageScanStatus{Status: ecrtypes.ScanStatusComplete},
			ImageScanFindings: &ecrtypes.ImageScanFindings{},
		}
		gomock.InOrder(
			client.EXPECT().DescribeImageScanFindings(t.Context(), input).Return(&ecr.DescribeImageScanFindingsOutput{
				ImageScanStatus: &ecrtypes.ImageScanStatus{Status: ecrtypes.ScanStatusInProgress},
			}, nil),
			client.EXPECT().DescribeImageScanFindings(t.Context(), input).Return(completed, nil),
		)
		got, err := w.WaitForImageScan(t.Context(), info, imageID, 30*time.Second)
		assert.NoError(t, err)
		assert.Same(t, completed, got)
	})
	for _, tc := range []struct {
		name    string
		status  *ecrtypes.ImageScanStatus
		wantErr string
	}{
		{name: "returns error for missing status", wantErr: "image scan status missing"},
		{name: "stops on failed scan", status: &ecrtypes.ImageScanStatus{Status: ecrtypes.ScanStatusFailed}, wantErr: "image scan failed: status=FAILED"},
		{name: "stops on unsupported image", status: &ecrtypes.ImageScanStatus{Status: ecrtypes.ScanStatusUnsupportedImage}, wantErr: "image scan failed: status=UNSUPPORTED_IMAGE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, client := setup(t, test.NewFakeTime())
			client.EXPECT().DescribeImageScanFindings(t.Context(), input).Return(&ecr.DescribeImageScanFindingsOutput{ImageScanStatus: tc.status}, nil)
			got, err := w.WaitForImageScan(t.Context(), info, imageID, 30*time.Second)
			assert.EqualError(t, err, tc.wantErr)
			assert.Nil(t, got)
		})
	}

	t.Run("returns describe error", func(t *testing.T) {
		w, client := setup(t, test.NewFakeTime())
		expected := errors.New("describe failed")
		client.EXPECT().DescribeImageScanFindings(t.Context(), input).Return(nil, expected)
		got, err := w.WaitForImageScan(t.Context(), info, imageID, 30*time.Second)
		assert.ErrorIs(t, err, expected)
		assert.Nil(t, got)
	})
	t.Run("stops when context is cancelled", func(t *testing.T) {
		w, _ := setup(t, test.NewNeverTimer())
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		got, err := w.WaitForImageScan(ctx, info, imageID, 30*time.Second)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, got)
	})
}
