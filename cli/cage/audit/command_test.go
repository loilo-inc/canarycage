package audit_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/smithy-go"
	"strings"
	"testing"

	"github.com/loilo-inc/canarycage/v5/cli/cage/audit"
	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/mocks/mock_audit"
	"github.com/loilo-inc/canarycage/v5/test"
	"github.com/loilo-inc/logos/v2/di"
	"github.com/stretchr/testify/assert"
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
			mockScanner.EXPECT().Scan(t.Context(), "cluster", "service").Return(nil, test.Err),
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

	t.Run("should return nil on successful scan", func(t *testing.T) {

		mockScanner, mockPrinter := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Printer, mockPrinter)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewNeverTimer())
		})

		var results []audit.ScanResult
		gomock.InOrder(
			mockScanner.EXPECT().Scan(t.Context(), "cluster", "service").Return(results, nil),
		)

		cmd := audit.NewCommand(mockDI, &cageapp.AuditCmdInput{
			Cluster: "cluster",
			Service: "service",
			App:     &cageapp.App{NoColor: true},
		})

		err := cmd.Run(t.Context())
		assert := assert.New(t)
		assert.NoError(err)
		assert.Len(mockPrinter.Stdout, 1) // only "No CVEs found" message (no header)
		assert.Contains(mockPrinter.Stdout[0], "No CVEs found")
		assert.Len(mockPrinter.Stderr, 1)
		assert.Equal(mockPrinter.Stderr[0], "\r")
	})

	t.Run("should log json output when JSON flag is set", func(t *testing.T) {

		mockScanner, mockPrinter := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Printer, mockPrinter)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewNeverTimer())
		})

		var results []audit.ScanResult
		gomock.InOrder(
			mockScanner.EXPECT().Scan(t.Context(), "cluster", "service").Return(results, nil),
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
		assert.NoError(err)
		assert.Len(mockPrinter.Stdout, 1)
		jsonOutput := mockPrinter.Stdout[0]
		var finalResult audit.FinalResult
		err = json.Unmarshal([]byte(jsonOutput), &finalResult)
		assert.NoError(err)
		assert.Equal("ap-northeast-1", finalResult.Target.Region)
		assert.Equal("cluster", finalResult.Target.Cluster)
		assert.Equal("service", finalResult.Target.Service)
		assert.Equal(0, finalResult.Result.Summary.CriticalCount)
		assert.Equal(0, finalResult.Result.Summary.HighCount)
		assert.Equal(0, finalResult.Result.Summary.MediumCount)
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
			mockScanner.EXPECT().Scan(ctx, "cluster", "service").DoAndReturn(func(context.Context, string, string) ([]*audit.ScanResult, error) {
				cancel()
				return nil, nil
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

func TestAuditCommandRun_ImageScanErrors(t *testing.T) {
	denied := &smithy.GenericAPIError{Code: "AccessDeniedException", Message: "not authorized to perform ecr:DescribeImageScanFindings"}
	failed := audit.ScanResult{ImageInfo: audit.ImageInfo{ContainerName: "app", Registry: "registry", Repository: "repo", Tag: "v1"}, Err: denied}
	clean := audit.ScanResult{ImageInfo: audit.ImageInfo{ContainerName: "sidecar", Registry: "registry", Repository: "sidecar", Tag: "v1"}}
	vulnerable := clean
	vulnerable.Cves = []audit.CVE{{Name: "CVE-TEST", Severity: ecrtypes.FindingSeverityHigh}}
	for _, tc := range []struct {
		name      string
		results   []audit.ScanResult
		wantErr   error
		wantCount int
	}{
		{name: "all images denied", results: []audit.ScanResult{failed}, wantErr: denied},
		{name: "denied alongside clean image", results: []audit.ScanResult{clean, failed}, wantErr: denied},
		{name: "denied alongside vulnerable image", results: []audit.ScanResult{vulnerable, failed}, wantErr: denied, wantCount: 1},
		{name: "scan not found", results: []audit.ScanResult{{ImageInfo: failed.ImageInfo, Err: audit.ErrScanNotFound}}, wantErr: audit.ErrScanNotFound},
		{name: "non-ECR image", results: []audit.ScanResult{{ImageInfo: failed.ImageInfo, Err: audit.ErrNonEcrImage}}, wantErr: audit.ErrNonEcrImage},
		{name: "successful scan without CVEs", results: []audit.ScanResult{clean}},
	} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%v", tc.name, jsonOutput), func(t *testing.T) {
				scanner := mock_audit.NewMockScanner(gomock.NewController(t))
				p := test.NewMockPrinter()
				d := di.NewDomain(func(b *di.B) {
					b.Set(key.Printer, p)
					b.Set(key.Scanner, scanner)
					b.Set(key.Time, test.NewNeverTimer())
				})
				scanner.EXPECT().Scan(t.Context(), "cluster", "service").Return(tc.results, nil)
				input := cageapp.NewAuditCmdInput()
				input.Cluster = "cluster"
				input.Service = "service"
				input.NoColor = true
				input.JSON = jsonOutput
				err := audit.NewCommand(d, input).Run(t.Context())
				if tc.wantErr != nil {
					assert.ErrorIs(t, err, tc.wantErr)
					_, reported := errors.AsType[*audit.ScanError](err)
					assert.True(t, reported)
					assert.Contains(t, err.Error(), "app")
					assert.NotContains(t, strings.Join(p.Stdout, ""), "No CVEs found")
				} else {
					assert.NoError(t, err)
				}
				if jsonOutput {
					var result audit.FinalResult
					if !assert.NoError(t, json.Unmarshal([]byte(strings.Join(p.Stdout, "")), &result)) {
						return
					}
					assert.Equal(t, tc.wantCount, result.Summary.TotalCount)
					if !assert.Len(t, result.ScanResults, len(tc.results)) {
						return
					}
					for i, scan := range result.ScanResults {
						assert.NotEmpty(t, scan.ContainerName)
						if i > 0 {
							assert.LessOrEqual(t, result.ScanResults[i-1].ContainerName, scan.ContainerName)
						}
						if scan.ContainerName == "app" {
							assert.Equal(t, tc.wantErr.Error(), scan.Error)
							if errors.Is(tc.wantErr, audit.ErrScanNotFound) {
								assert.Equal(t, audit.ScanStatusNA, scan.Status)
							} else {
								assert.Equal(t, audit.ScanStatusError, scan.Status)
							}
						} else {
							assert.Empty(t, scan.Error)
						}
					}
				} else if tc.wantErr != nil {
					assert.Contains(t, strings.Join(p.Stderr, ""), tc.wantErr.Error())
					assert.Contains(t, strings.Join(p.Stdout, ""), "CONTAINER")
					assert.True(t, strings.HasPrefix(p.Logs[len(p.Logs)-1], "|"), "table must be the final output")
					if tc.wantCount > 0 {
						assert.Contains(t, strings.Join(p.Stdout, ""), "CVE-TEST")
					}
				} else {
					assert.Contains(t, strings.Join(p.Stdout, ""), "No CVEs found")
				}
			})
		}
	}
}
