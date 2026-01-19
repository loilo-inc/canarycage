package commands

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"

	"github.com/loilo-inc/canarycage/cli/cage/scan"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_logger"
	"github.com/loilo-inc/canarycage/mocks/mock_scan"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"
)

func TestScan(t *testing.T) {
	t.Run("returns error when both directory and flags are missing", func(t *testing.T) {
		app := setupScanApp(t, nil)

		err := app.Run([]string{"cage", "scan", "--region", "us-east-1"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either directory argument or both --cluster and --service flags must be provided")
	})

	t.Run("returns error when only cluster flag is provided", func(t *testing.T) {
		app := setupScanApp(t, nil)

		err := app.Run([]string{"cage", "scan", "--region", "us-east-1", "--cluster", "test-cluster"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either directory argument or both --cluster and --service flags must be provided")
	})

	t.Run("returns error when only service flag is provided", func(t *testing.T) {
		app := setupScanApp(t, nil)

		err := app.Run([]string{"cage", "scan", "--region", "us-east-1", "--service", "test-service"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either directory argument or both --cluster and --service flags must be provided")
	})

	t.Run("returns error when diProvider fails", func(t *testing.T) {
		expectedErr := errors.New("di provider error")
		app := setupScanApp(t, func(region string) (*di.D, error) {
			return nil, expectedErr
		})

		err := app.Run([]string{
			"cage", "scan", "--region", "us-east-1", "--cluster", "test-cluster", "--service", "test-service",
		})
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestScan_Success(t *testing.T) {
	setup := func(t *testing.T) *cli.App {
		t.Helper()
		ctrl := gomock.NewController(t)
		mockScanner := mock_scan.NewMockScanner(ctrl)
		mockLogger := mock_logger.NewMockLogger(ctrl)
		d := di.NewDomain(func(b *di.B) {
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Logger, mockLogger)
		})

		app := setupScanApp(t, func(region string) (*di.D, error) {
			assert.Equal(t, "us-east-1", region)
			return d, nil
		})

		mockScanner.EXPECT().
			Scan(gomock.Any(), "cluster", "service"). // from fixtures/service.json
			Return(makeScanResult(), nil)

		mockLogger.EXPECT().Printf(
			gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(),
		).Times(2)

		return app
	}
	t.Run("executes scan with directory argument", func(t *testing.T) {
		app := setup(t)
		err := app.Run([]string{"cage", "scan",
			"--region", "us-east-1", "../../../fixtures"})
		assert.NoError(t, err)
	})
	t.Run("executes scan with flags", func(t *testing.T) {
		app := setup(t)
		err := app.Run([]string{"cage", "scan",
			"--region", "us-east-1",
			"--cluster", "cluster",
			"--service", "service"})
		assert.NoError(t, err)
	})
}

func setupScanApp(t *testing.T, diProvider func(region string) (*di.D, error)) *cli.App {
	t.Helper()
	app := cli.NewApp()
	app.Name = "cage"
	app.Commands = []*cli.Command{
		Scan(diProvider),
	}
	return app
}

func makeScanResult() []*scan.ScanResult {
	return []*scan.ScanResult{
		{
			ImageInfo: &scan.ImageInfo{
				Repository:    "test-repo",
				Tag:           "latest",
				Registry:      "dockerhub.io",
				ContainerName: "web-app",
				PlatformArch:  "amd64",
			},
			ImageScanFindings: &ecrtypes.ImageScanFindings{
				Findings: []ecrtypes.ImageScanFinding{
					{
						Name:        aws.String("CVE-2023-1234"),
						Description: aws.String("Test vulnerability description"),
						Severity:    ecrtypes.FindingSeverityHigh,
					},
				},
			},
		},
	}
}
