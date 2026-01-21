package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/mocks/mock_types"
	"github.com/loilo-inc/canarycage/types"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"
)

func TestAudit(t *testing.T) {
	t.Run("returns error when region is missing", func(t *testing.T) {
		app := setupAuditApp(t, nil)
		err := app.Run([]string{"cage", "audit", "--region", ""})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--region flag is required")
	})
	t.Run("return errors when too many arguments", func(t *testing.T) {
		app := setupAuditApp(t, nil)
		err := app.Run([]string{"cage", "audit", "--region", "us-east-1", "arg1", "arg2"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid number of arguments. expected at most 1")
	})
	t.Run("returns error when both directory and flags are missing", func(t *testing.T) {
		app := setupAuditApp(t, nil)

		err := app.Run([]string{"cage", "audit", "--region", "us-east-1"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either directory argument or both --cluster and --service flags must be provided")
	})

	t.Run("returns error when only cluster flag is provided", func(t *testing.T) {
		app := setupAuditApp(t, nil)

		err := app.Run([]string{"cage", "audit", "--region", "us-east-1", "--cluster", "test-cluster"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either directory argument or both --cluster and --service flags must be provided")
	})

	t.Run("returns error when only service flag is provided", func(t *testing.T) {
		app := setupAuditApp(t, nil)

		err := app.Run([]string{"cage", "audit", "--region", "us-east-1", "--service", "test-service"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either directory argument or both --cluster and --service flags must be provided")
	})

	t.Run("returns error when diProvider fails", func(t *testing.T) {
		expectedErr := errors.New("di provider error")
		app := setupAuditApp(t, func(ctx context.Context, input *cageapp.AuditCmdInput) (types.Audit, error) {
			assert.Equal(t, "us-east-1", input.Region)
			return nil, expectedErr
		})

		err := app.Run([]string{
			"cage", "audit", "--region", "us-east-1", "--cluster", "test-cluster", "--service", "test-service",
		})
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
	setupBase := func(t *testing.T) (*cli.App, *mock_types.MockAudit) {
		t.Helper()
		ctrl := gomock.NewController(t)
		mockAudit := mock_types.NewMockAudit(ctrl)

		app := setupAuditApp(t, func(ctx context.Context, input *cageapp.AuditCmdInput) (types.Audit, error) {
			assert.Equal(t, "us-east-1", input.Region)
			return mockAudit, nil
		})
		return app, mockAudit
	}
	t.Run("Succcess", func(t *testing.T) {
		setup := func(t *testing.T) *cli.App {
			t.Helper()
			app, mockAudit := setupBase(t)
			mockAudit.EXPECT().
				Run(gomock.Any()).
				Return(nil)
			return app
		}
		t.Run("executes scan with directory argument", func(t *testing.T) {
			app := setup(t)
			err := app.Run([]string{"cage", "audit",
				"--region", "us-east-1", "../../../fixtures"})
			assert.NoError(t, err)
		})
		t.Run("executes scan with flags", func(t *testing.T) {
			app := setup(t)
			err := app.Run([]string{"cage", "audit",
				"--region", "us-east-1",
				"--cluster", "cluster",
				"--service", "service"})
			assert.NoError(t, err)
		})
	})
	t.Run("Error", func(t *testing.T) {
		t.Run("error on scanner.Scan()", func(t *testing.T) {
			app, mockAudit := setupBase(t)
			mockAudit.EXPECT().
				Run(gomock.Any()).
				Return(errors.New("scan error"))

			err := app.Run([]string{"cage", "audit",
				"--region", "us-east-1",
				"--cluster", "cluster",
				"--service", "service"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "scan error")
		})
		t.Run("error on loading service definition", func(t *testing.T) {
			app := setupAuditApp(t, nil)
			err := app.Run([]string{"cage", "audit",
				"--region", "us-east-1", "../../../fixtures/invalid-service"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no 'service.json' found")
		})
	})
}

func setupAuditApp(t *testing.T, provider cageapp.AuditCmdProvider) *cli.App {
	t.Helper()
	conf := &cageapp.App{}
	app := cli.NewApp()
	app.Name = "cage"
	app.Commands = []*cli.Command{
		Audit(conf, provider),
	}
	return app
}
