package cage

import (
	"context"
	"time"

	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/timeout"
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

type cage struct {
	*Input
	Timeout timeout.Manager
}

type Input struct {
	Env  *Envars
	Ecs  awsiface.EcsClient
	Alb  awsiface.AlbClient
	Ec2  awsiface.Ec2Client
	Time Time
}

func NewCage(input *Input) Cage {
	if input.Time == nil {
		input.Time = &timeImpl{}
	}
	taskRunningWait := (time.Duration)(input.Env.CanaryTaskRunningWait) * time.Second
	taskHealthCheckWait := (time.Duration)(input.Env.CanaryTaskHealthCheckWait) * time.Second
	taskStoppedWait := (time.Duration)(input.Env.CanaryTaskStoppedWait) * time.Second
	serviceStableWait := (time.Duration)(input.Env.ServiceStableWait) * time.Second
	return &cage{
		Input: input,
		Timeout: timeout.NewManager(
			10*time.Minute,
			&timeout.Input{
				TaskRunningWait:     taskRunningWait,
				TaskHealthCheckWait: taskHealthCheckWait,
				TaskStoppedWait:     taskStoppedWait,
				ServiceStableWait:   serviceStableWait,
			}),
	}
}
