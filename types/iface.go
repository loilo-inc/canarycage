package types

import (
	"context"
	"time"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
)

type Cage interface {
	Up(ctx context.Context) (*UpResult, error)
	Run(ctx context.Context, input *RunInput) (*RunResult, error)
	RollOut(ctx context.Context, input *RollOutInput) (*RollOutResult, error)
}

type Time interface {
	Now() time.Time
	NewTimer(time.Duration) *time.Timer
}

type Deps struct {
	Env  *env.Envars
	Ecs  awsiface.EcsClient
	Alb  awsiface.AlbClient
	Ec2  awsiface.Ec2Client
	Srv  awsiface.SrvClient
	Time Time
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
