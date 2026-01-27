package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/types"
	"github.com/stretchr/testify/assert"
)

type fakeCage struct{}

func (f *fakeCage) Up(context.Context) (*types.UpResult, error) {
	return nil, nil
}

func (f *fakeCage) Run(context.Context, *types.RunInput) (*types.RunResult, error) {
	return nil, nil
}

func (f *fakeCage) RollOut(context.Context, *types.RollOutInput) (*types.RollOutResult, error) {
	return nil, nil
}

func TestSetupCage(t *testing.T) {
	t.Run("loads definitions and returns cage", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"

		expected := &fakeCage{}
		var providerCalled bool
		cmds := NewCageCommands(func(ctx context.Context, got *cageapp.CageCmdInput) (types.Cage, error) {
			providerCalled = true
			assert.Same(t, input, got)
			assert.Equal(t, "cluster", got.Cluster)
			assert.Equal(t, "service", got.Service)
			assert.NotNil(t, got.ServiceDefinitionInput)
			assert.NotNil(t, got.TaskDefinitionInput)
			return expected, nil
		})

		cagecli, err := cmds.setupCage(input, "../../../fixtures")
		assert.NoError(t, err)
		assert.True(t, providerCalled)
		assert.Same(t, expected, cagecli)
	})

	t.Run("skips task definition when arn is provided", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"
		input.TaskDefinitionArn = "arn://task"

		dir := t.TempDir()
		serviceJSON, err := os.ReadFile("../../../fixtures/service.json")
		if err != nil {
			t.Fatalf("failed to read fixture: %s", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "service.json"), serviceJSON, 0o644); err != nil {
			t.Fatalf("failed to write service.json: %s", err)
		}

		expected := &fakeCage{}
		cmds := NewCageCommands(func(ctx context.Context, got *cageapp.CageCmdInput) (types.Cage, error) {
			assert.Equal(t, "cluster", got.Cluster)
			assert.Equal(t, "service", got.Service)
			assert.NotNil(t, got.ServiceDefinitionInput)
			assert.Nil(t, got.TaskDefinitionInput)
			return expected, nil
		})

		cagecli, err := cmds.setupCage(input, dir)
		assert.NoError(t, err)
		assert.Same(t, expected, cagecli)
	})

	t.Run("returns error when service definition is missing", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"
		input.TaskDefinitionArn = "arn://task"

		dir := t.TempDir()
		cmds := NewCageCommands(func(context.Context, *cageapp.CageCmdInput) (types.Cage, error) {
			t.Fatal("provider should not be called when service.json is missing")
			return nil, nil
		})

		_, err := cmds.setupCage(input, dir)
		assert.EqualError(t, err, "no 'service.json' found in "+dir)
	})

	t.Run("returns error when task definition is missing", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"

		dir := t.TempDir()
		serviceJSON, err := os.ReadFile("../../../fixtures/service.json")
		if err != nil {
			t.Fatalf("failed to read fixture: %s", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "service.json"), serviceJSON, 0o644); err != nil {
			t.Fatalf("failed to write service.json: %s", err)
		}

		cmds := NewCageCommands(func(context.Context, *cageapp.CageCmdInput) (types.Cage, error) {
			t.Fatal("provider should not be called when task-definition.json is missing")
			return nil, nil
		})

		_, err = cmds.setupCage(input, dir)
		assert.EqualError(t, err, "no 'task-definition.json' found in "+dir)
	})

	t.Run("returns error when required envars are missing", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.TaskDefinitionArn = "arn://task"

		cmds := NewCageCommands(func(context.Context, *cageapp.CageCmdInput) (types.Cage, error) {
			t.Fatal("provider should not be called when envars are invalid")
			return nil, nil
		})

		_, err := cmds.setupCage(input, "../../../fixtures")
		assert.EqualError(t, err, "--region [CAGE_REGION] is required")
	})

	t.Run("propagates provider error", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Region = "us-west-2"
		input.TaskDefinitionArn = "arn://task"

		expectedErr := errors.New("provider error")
		cmds := NewCageCommands(func(ctx context.Context, got *cageapp.CageCmdInput) (types.Cage, error) {
			assert.Equal(t, "cluster", got.Cluster)
			assert.Equal(t, "service", got.Service)
			return nil, expectedErr
		})

		_, err := cmds.setupCage(input, "../../../fixtures")
		assert.EqualError(t, err, "provider error")
	})
}
