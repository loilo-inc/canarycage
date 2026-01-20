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
	makeMockLogger := func(ctrl *gomock.Controller) *mock_logger.MockLogger {
		mockLogger := mock_logger.NewMockLogger(ctrl)
		mockLogger.EXPECT().Printf(gomock.Any(), gomock.Any()).AnyTimes()
		mockLogger.EXPECT().Printf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockLogger.EXPECT().Printf(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		return mockLogger
	}
	t.Run("should return error from scanner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ctx := context.Background()
		mockScanner := mock_audit.NewMockScanner(ctrl)

		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, makeMockLogger(ctrl))
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewFakeNeverTimer())
		})

		mockScanner.EXPECT().Scan(ctx, "cluster", "service").Return(nil, test.Err)

		app := &cageapp.App{NoColor: false}
		cmd := audit.NewCommand(mockDI, app, false)

		err := cmd.Run(ctx, "cluster", "service")
		assert.EqualError(t, err, "error")
	})

	t.Run("should return nil on successful scan", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ctx := context.Background()
		mockScanner := mock_audit.NewMockScanner(ctrl)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, makeMockLogger(ctrl))
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewFakeNeverTimer())
		})

		results := []*audit.ScanResult{}
		mockScanner.EXPECT().Scan(ctx, "cluster", "service").Return(results, nil)

		app := &cageapp.App{NoColor: false}
		cmd := audit.NewCommand(mockDI, app, false)

		err := cmd.Run(ctx, "cluster", "service")
		assert.NoError(t, err)
	})

	t.Run("should return context error when context is cancelled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		ctx, cancel := context.WithCancel(context.Background())
		mockScanner := mock_audit.NewMockScanner(ctrl)
		mockDI := di.NewDomain(func(b *di.B) {
			b.Set(key.Logger, makeMockLogger(ctrl))
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Time, test.NewFakeNeverTimer())
		})
		mockScanner.EXPECT().Scan(ctx, "cluster", "service").DoAndReturn(func(context.Context, string, string) ([]audit.ScanResult, error) {
			cancel()
			return nil, nil
		})

		app := &cageapp.App{NoColor: false}
		cmd := audit.NewCommand(mockDI, app, false)

		err := cmd.Run(ctx, "cluster", "service")
		assert.Equal(t, context.Canceled, err)
	})
}
