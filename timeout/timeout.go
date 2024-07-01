package timeout

import (
	"time"

	"github.com/loilo-inc/canarycage/env"
)

type Manager interface {
	TaskRunning() time.Duration
	TaskHealthCheck() time.Duration
	TaskStopped() time.Duration
	ServiceStable() time.Duration
	TargetHealthCheck() time.Duration
}

type manager struct {
	env            *env.Envars
	DefaultTimeout time.Duration
}

func NewManager(
	env *env.Envars,
	defaultTimeout time.Duration,
) Manager {
	return &manager{
		env:            env,
		DefaultTimeout: defaultTimeout,
	}
}

func (t *manager) TaskRunning() time.Duration {
	wait := t.env.CanaryTaskRunningWait
	if wait > 0 {
		return time.Duration(wait) * time.Second
	}
	return t.DefaultTimeout
}

func (t *manager) TaskHealthCheck() time.Duration {
	wait := t.env.CanaryTaskHealthCheckWait
	if wait > 0 {
		return time.Duration(wait) * time.Second
	}
	return t.DefaultTimeout
}

func (t *manager) TaskStopped() time.Duration {
	wait := t.env.CanaryTaskStoppedWait
	if wait > 0 {
		return time.Duration(wait) * time.Second
	}
	return t.DefaultTimeout
}

func (t *manager) ServiceStable() time.Duration {
	wait := t.env.ServiceStableWait
	if wait > 0 {
		return time.Duration(wait) * time.Second
	}
	return t.DefaultTimeout
}

func (t *manager) TargetHealthCheck() time.Duration {
	wait := t.env.TargetHealthCheckWait
	if wait > 0 {
		return time.Duration(wait) * time.Second
	}
	return t.DefaultTimeout
}
