package audit_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/audit"
	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_audit"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuditCommandRun(t *testing.T) {
	setup := func(t *testing.T) (*mock_audit.MockScanner, *test.MockLogger) {
		t.Helper()
		ctrl := gomock.NewController(t)
		mockScanner := mock_audit.NewMockScanner(ctrl)
		mockLogger := &test.MockLogger{}
		return mockScanner, mockLogger
	}
	t.Run("should return error from scanner", func(t *testing.T) {
		ctx := context.Background()
		mockScanner, mockLogger := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, mockLogger)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewNeverTimer())
		})
		gomock.InOrder(
			mockScanner.EXPECT().Scan(ctx, "cluster", "service").Return(nil, test.Err),
		)
		input := cageapp.NewAuditCmdInput()
		input.Cluster = "cluster"
		input.Service = "service"
		cmd := audit.NewCommand(mockDI, input)
		err := cmd.Run(ctx)
		assert := assert.New(t)
		assert.Equal(test.Err, err)
		assert.Len(mockLogger.Stderr, 1)
	})

	t.Run("should return nil on successful scan", func(t *testing.T) {
		ctx := context.Background()

		mockScanner, mockLogger := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, mockLogger)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewNeverTimer())
		})

		var results []*audit.ScanResult
		gomock.InOrder(
			mockScanner.EXPECT().Scan(ctx, "cluster", "service").Return(results, nil),
		)

		cmd := audit.NewCommand(mockDI, &cageapp.AuditCmdInput{
			Cluster: "cluster",
			Service: "service",
			App:     &cageapp.App{NoColor: true},
		})

		err := cmd.Run(ctx)
		assert := assert.New(t)
		assert.NoError(err)
		assert.Len(mockLogger.Stdout, 2) // header + no findings
		assert.Contains(mockLogger.Stdout[0], "CONTAINER")
		assert.Contains(mockLogger.Stdout[1], "No CVEs found")
		assert.Len(mockLogger.Stderr, 1)
		assert.Equal(mockLogger.Stderr[0], "\r")
	})

	t.Run("should log json output when JSON flag is set", func(t *testing.T) {
		ctx := context.Background()

		mockScanner, mockLogger := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, mockLogger)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewNeverTimer())
		})

		var results []*audit.ScanResult
		gomock.InOrder(
			mockScanner.EXPECT().Scan(ctx, "cluster", "service").Return(results, nil),
		)

		cmd := audit.NewCommand(mockDI, &cageapp.AuditCmdInput{
			Cluster: "cluster",
			Service: "service",
			JSON:    true,
			App:     &cageapp.App{NoColor: true},
		})

		err := cmd.Run(ctx)
		assert := assert.New(t)
		assert.NoError(err)
		assert.Len(mockLogger.Stdout, 1)
		jsonOutput := mockLogger.Stdout[0]
		var finalResult audit.FinalResult
		err = json.Unmarshal([]byte(jsonOutput), &finalResult)
		assert.NoError(err)
		assert.Equal("cluster", finalResult.Target.Cluster)
		assert.Equal("service", finalResult.Target.Service)
		assert.Equal(0, finalResult.Result.Summary.CriticalCount)
		assert.Equal(0, finalResult.Result.Summary.HighCount)
		assert.Equal(0, finalResult.Result.Summary.MediumCount)
		// only the spinner removal log
		assert.Len(mockLogger.Stderr, 1)
		assert.Equal(mockLogger.Stderr[0], "\r")
	})

	t.Run("should return context error when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mockScanner, mockLogger := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, mockLogger)
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
