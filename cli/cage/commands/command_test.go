package commands

import (
	"context"
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

func TestAuditCommand_Run(t *testing.T) {
	t.Run("should return error when doScan fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockScanner := mock_audit.NewMockScanner(ctrl)
		mockScanner.EXPECT().
			Scan(gomock.Any(), "test-cluster", "test-service").
			Return(nil, test.Err)

		di := di.NewDomain(func(b *di.B) {
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Logger, &test.MockLogger{})
			b.Set(key.Time, test.NewNeverTimer())
		})

		input := &cageapp.AuditCmdInput{
			Cluster: "test-cluster",
			Service: "test-service",
		}

		cmd := audit.NewCommand(di, input)
		err := cmd.Run(context.Background())
		assert.EqualError(t, err, "error")
	})

	t.Run("should print results in text format", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockResults := []*audit.ScanResult{
			{ /* mock result data */ },
		}

		mockScanner := mock_audit.NewMockScanner(ctrl)
		mockScanner.EXPECT().
			Scan(gomock.Any(), "test-cluster", "test-service").
			Return(mockResults, nil)

		di := di.NewDomain(func(b *di.B) {
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Logger, &test.MockLogger{})
			b.Set(key.Time, test.NewNeverTimer())
		})

		input := &cageapp.AuditCmdInput{
			App:       &cageapp.App{NoColor: false},
			Cluster:   "test-cluster",
			Service:   "test-service",
			LogDetail: false,
			JSON:      false,
		}

		cmd := audit.NewCommand(di, input)
		err := cmd.Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("should print results in JSON format", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockResults := []*audit.ScanResult{
			{ /* mock result data */ },
		}

		mockScanner := mock_audit.NewMockScanner(ctrl)
		mockScanner.EXPECT().
			Scan(gomock.Any(), "test-cluster", "test-service").
			Return(mockResults, nil)

		di := di.NewDomain(func(b *di.B) {
			b.Set(key.Scanner, mockScanner)
			b.Set(key.Logger, &test.MockLogger{})
			b.Set(key.Time, test.NewNeverTimer())
		})

		input := &cageapp.AuditCmdInput{
			App:       &cageapp.App{NoColor: true},
			Region:    "us-west-2",
			Cluster:   "test-cluster",
			Service:   "test-service",
			LogDetail: true,
			JSON:      true,
		}

		cmd := audit.NewCommand(di, input)
		err := cmd.Run(context.Background())
		assert.NoError(t, err)
	})
}
