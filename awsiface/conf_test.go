package awsiface

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
)

func TestMustLoadConfig_Success(t *testing.T) {
	ctx := context.Background()

	// This should not panic in normal circumstances
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustLoadConfig panicked unexpectedly: %v", r)
		}
	}()

	cfg := MustLoadConfig(ctx)

	if cfg.Region == "" && cfg.Credentials == nil {
		t.Log("Config loaded (region or credentials may be empty in test environment)")
	}
}

func TestMustLoadConfig_WithOptions(t *testing.T) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustLoadConfig with options panicked unexpectedly: %v", r)
		}
	}()

	cfg := MustLoadConfig(ctx, config.WithRegion("us-west-2"))

	if cfg.Region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got %s", cfg.Region)
	}
}

func TestMustLoadConfig_Panic(t *testing.T) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustLoadConfig to panic with invalid option, but it didn't")
		}
	}()

	// Pass an option that returns an error to trigger panic
	invalidOpt := func(*config.LoadOptions) error {
		return errors.New("forced error")
	}

	MustLoadConfig(ctx, invalidOpt)
}
