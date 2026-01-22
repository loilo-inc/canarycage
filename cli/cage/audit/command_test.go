package audit_test

import (
	"context"
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/audit"
	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/mocks/mock_audit"
	"github.com/loilo-inc/canarycage/mocks/mock_logger"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuditCommandRun(t *testing.T) {
	setup := func(t *testing.T) (*mock_audit.MockScanner, *mock_logger.MockLogger, *mock_audit.MockPrinter) {
		t.Helper()
		ctrl := gomock.NewController(t)
		mockScanner := mock_audit.NewMockScanner(ctrl)
		mockPrinter := mock_audit.NewMockPrinter(ctrl)
		mockLogger := mock_logger.NewMockLogger(ctrl)
		return mockScanner, mockLogger, mockPrinter
	}
	t.Run("should return error from scanner", func(t *testing.T) {
		ctx := context.Background()
		mockScanner, mockLogger, _ := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, mockLogger)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewFakeNeverTimer())
		})
		gomock.InOrder(
			mockScanner.EXPECT().Scan(ctx, "cluster", "service").Return(nil, test.Err),
			mockLogger.EXPECT().Printf("\r"),
		)
		input := cageapp.NewAuditCmdInput()
		input.Cluster = "cluster"
		input.Service = "service"
		cmd := audit.NewCommand(mockDI, input)
		err := cmd.Run(ctx)
		assert.Equal(t, test.Err, err)
	})

	t.Run("should return nil on successful scan", func(t *testing.T) {
		ctx := context.Background()

		mockScanner, mockLogger, mockPrinter := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, mockLogger)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Printer, mockPrinter)
			b.Set(key.Time, test.NewFakeNeverTimer())
		})

		var results []*audit.ScanResult
		gomock.InOrder(
			mockScanner.EXPECT().Scan(ctx, "cluster", "service").Return(results, nil),
			mockLogger.EXPECT().Printf("\r"),
			mockPrinter.EXPECT().Print(results),
		)

		cmd := audit.NewCommand(mockDI, &cageapp.AuditCmdInput{
			Cluster: "cluster",
			Service: "service",
			App:     &cageapp.App{},
		})

		err := cmd.Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("should return context error when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		mockScanner, mockLogger, _ := setup(t)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, mockLogger)
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewFakeNeverTimer())
		})
		gomock.InOrder(
			mockScanner.EXPECT().Scan(ctx, "cluster", "service").DoAndReturn(func(context.Context, string, string) ([]audit.ScanResult, error) {
				cancel()
				return nil, nil
			}),
			mockLogger.EXPECT().Printf("\r"),
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
