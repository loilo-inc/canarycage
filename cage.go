package cage

import (
	"time"

	"github.com/loilo-inc/canarycage/timeout"
	"github.com/loilo-inc/canarycage/types"
)

type cage struct {
	*types.Deps
	Timeout timeout.Manager
}

func NewCage(input *types.Deps) types.Cage {
	if input.Time == nil {
		input.Time = &timeImpl{}
	}
	taskRunningWait := (time.Duration)(input.Env.CanaryTaskRunningWait) * time.Second
	taskHealthCheckWait := (time.Duration)(input.Env.CanaryTaskHealthCheckWait) * time.Second
	taskStoppedWait := (time.Duration)(input.Env.CanaryTaskStoppedWait) * time.Second
	serviceStableWait := (time.Duration)(input.Env.ServiceStableWait) * time.Second
	return &cage{
		Deps: input,
		Timeout: timeout.NewManager(
			15*time.Minute,
			&timeout.Input{
				TaskRunningWait:     taskRunningWait,
				TaskHealthCheckWait: taskHealthCheckWait,
				TaskStoppedWait:     taskStoppedWait,
				ServiceStableWait:   serviceStableWait,
			}),
	}
}
