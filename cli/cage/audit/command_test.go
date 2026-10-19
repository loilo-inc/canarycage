package audit_test

import (
	"context"
	"encoding/json"
	"testing"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/loilo-inc/canarycage/v5/cli/cage/audit"
	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/mocks/mock_audit"
	"github.com/loilo-inc/canarycage/v5/test"
	"github.com/loilo-inc/logos/v2/di"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuditCommandRun(t *testing.T) {
	setup := func(t *testing.T) (*mock_audit.MockScanner, *test.MockPrinter) {
		t.Helper()
		ctrl := gomock.NewController(t)
		mockScanner := mock_audit.NewMockScanner(ctrl)
		mockPrinter := &test.MockPrinter{}
		return mockScanner, mockPrinter
	}
	t.Run("should return error from scanner", func(t *testing.T) {
		mockScanner, mockPrinter := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Printer, mockPrinter)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewNeverTimer())
		})
		gomock.InOrder(
			mockScanner.EXPECT().Scan(t.Context(), audit.ScanRequest{Cluster: "cluster", Service: "service"}).Return(nil, test.Err),
		)
		input := cageapp.NewAuditCmdInput()
		input.Cluster = "cluster"
		input.Service = "service"
		cmd := audit.NewCommand(mockDI, input)
		err := cmd.Run(t.Context())
		assert := assert.New(t)
		assert.Equal(test.Err, err)
		assert.Len(mockPrinter.Stderr, 1)
	})

	for _, inPlace := range []bool{false, true} {
		name := "should return nil on successful scan"
		if inPlace {
			name = "should forward in-place scan request"
		}
		t.Run(name, func(t *testing.T) {

			mockScanner, mockPrinter := setup(t)
			mockDI := di.NewDomain(func(b *di.B) {
				b.Set(key.Printer, mockPrinter)
				b.Set(key.Scanner, mockScanner)
				b.Set(key.Time, test.NewNeverTimer())
			})

			var results []audit.ScanResult
			gomock.InOrder(
				mockScanner.EXPECT().Scan(t.Context(), audit.ScanRequest{Cluster: "cluster", Service: "service", InPlace: inPlace}).Return(results, nil),
			)

			cmd := audit.NewCommand(mockDI, &cageapp.AuditCmdInput{
				InPlace: inPlace,
				Cluster: "cluster",
				Service: "service",
				App:     &cageapp.App{NoColor: true},
			})

			err := cmd.Run(t.Context())
			assert := assert.New(t)
			assert.NoError(err)
			if assert.Len(mockPrinter.Stdout, 2) {
				assert.Contains(mockPrinter.Stdout[0], "CONTAINER")
				assert.Equal("\nNo CVEs found\n", mockPrinter.Stdout[1])
			}
			assert.Len(mockPrinter.Stderr, 1)
			assert.Equal(mockPrinter.Stderr[0], "\r")
		})

	}

	t.Run("should log json output when JSON flag is set", func(t *testing.T) {

		mockScanner, mockPrinter := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Printer, mockPrinter)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewNeverTimer())
		})

		results := []audit.ScanResult{
			{
				ImageInfo: audit.ImageInfo{
					ContainerName: "app",
					Registry:      "123456789012.dkr.ecr.ap-northeast-1.amazonaws.com",
					Repository:    "my-app",
					Tag:           "v1.2.3",
					Digest:        "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					PlatformOS:    audit.PlatformOSLinux,
					PlatformArch:  audit.PlatformArchArm64,
				},
				Cves: []audit.CVE{
					{
						Name: "CVE-2025-0001", Description: "Critical vulnerability in libssl",
						PackageName: "libssl3", PackageVersion: "3.0.11",
						Uri: "https://example.com/CVE-2025-0001", Severity: ecrtypes.FindingSeverityCritical,
					},
					{
						Name: "CVE-2025-0002", Description: "High severity vulnerability in curl",
						PackageName: "curl", PackageVersion: "8.4.0",
						Uri: "https://example.com/CVE-2025-0002", Severity: ecrtypes.FindingSeverityHigh,
					},
					{
						Name: "CVE-2025-0003", Description: "Medium severity vulnerability in zlib",
						PackageName: "zlib", PackageVersion: "1.2.13",
						Uri: "https://example.com/CVE-2025-0003", Severity: ecrtypes.FindingSeverityMedium,
					},
				},
			},
			{
				ImageInfo: audit.ImageInfo{
					ContainerName: "sidecar",
					Registry:      "docker.io",
					Repository:    "library/nginx",
					Tag:           "1.27",
					PlatformOS:    audit.PlatformOSLinux,
					PlatformArch:  audit.PlatformArchArm64,
				},
				Err: audit.ErrNonEcrImage,
			},
		}
		gomock.InOrder(
			mockScanner.EXPECT().Scan(t.Context(), audit.ScanRequest{Cluster: "cluster", Service: "service"}).Return(results, nil),
		)

		cmd := audit.NewCommand(mockDI, &cageapp.AuditCmdInput{
			Region:  "ap-northeast-1",
			Cluster: "cluster",
			Service: "service",
			JSON:    true,
			App:     &cageapp.App{NoColor: true},
		})

		err := cmd.Run(t.Context())
		assert := assert.New(t)
		require.NoError(t, err)
		require.Len(t, mockPrinter.Stdout, 1)
		exp := audit.FinalResult{
			Target: audit.Target{
				Region:  "ap-northeast-1",
				Cluster: "cluster",
				Service: "service",
				Images:  []audit.ImageInfo{results[0].ImageInfo, results[1].ImageInfo},
			},
			Result: audit.Result{
				Summary: &audit.AggregateResult{
					CriticalCount: 1, HighCount: 1, MediumCount: 1,
					TotalCount: 3, HighestSeverity: ecrtypes.FindingSeverityCritical,
				},
				Vulns: []audit.Vuln{
					{CVE: results[0].Cves[0], Containers: []string{"app"}},
					{CVE: results[0].Cves[1], Containers: []string{"app"}},
					{CVE: results[0].Cves[2], Containers: []string{"app"}},
				},
				Errors: []audit.ScanError{
					{ContainerName: "sidecar", ErrorMessage: "non-ECR image"},
				},
			},
			ScannedAt: "0001-01-01T00:00:00Z",
		}
		expJSON, err := json.Marshal(exp)
		require.NoError(t, err)
		assert.JSONEq(string(expJSON), mockPrinter.Stdout[0])
		// only the spinner removal log
		assert.Len(mockPrinter.Stderr, 1)
		assert.Equal(mockPrinter.Stderr[0], "\r")
	})

	t.Run("should return context error when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mockScanner, mockPrinter := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Printer, mockPrinter)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewNeverTimer())
		})
		gomock.InOrder(
			mockScanner.EXPECT().Scan(ctx, audit.ScanRequest{Cluster: "cluster", Service: "service"}).DoAndReturn(func(context.Context, audit.ScanRequest) ([]audit.ScanResult, error) {
				cancel()
				return nil, ctx.Err()
			}),
		)

		cmd := audit.NewCommand(mockDI, &cageapp.AuditCmdInput{
			Cluster: "cluster",
			Service: "service",
			App:     &cageapp.App{},
		})

		err := cmd.Run(ctx)
		assert.Equal(t, context.Canceled, err)
	})
}
