package types

import (
	"context"
	"time"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type Cage interface {
	Up(ctx context.Context) (*UpResult, error)
	Run(ctx context.Context, input *RunInput) (*RunResult, error)
	RollOut(ctx context.Context, input *RollOutInput) (*RollOutResult, error)
}

type Time interface {
	NewTimer(time.Duration) *time.Timer
}

type RunInput struct {
	Container *string
	Overrides *ecstypes.TaskOverride
}

type RunResult struct {
	ExitCode int32
}

type RollOutInput struct {
	// UpdateService is a flag to update service with changed configurations except for task definition
	UpdateService bool
}

type RollOutResult struct {
	ServiceIntact bool
}

type UpResult struct {
	TaskDefinition *ecstypes.TaskDefinition
	Service        *ecstypes.Service
}
