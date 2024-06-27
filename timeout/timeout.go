package timeout

import "time"

type Input struct {
	TaskStoppedWait     time.Duration
	TaskRunningWait     time.Duration
	TaskHealthCheckWait time.Duration
	ServiceStableWait   time.Duration
}

type Manager interface {
	TaskRunning() time.Duration
	TaskHealthCheck() time.Duration
	TaskStopped() time.Duration
	ServiceStable() time.Duration
}

type manager struct {
	*Input
	DefaultTimeout time.Duration
}

func NewManager(
	defaultTimeout time.Duration,
	input *Input,
) Manager {
	return &manager{
		Input:          input,
		DefaultTimeout: defaultTimeout,
	}
}

func (t *manager) TaskRunning() time.Duration {
	if t.TaskRunningWait > 0 {
		return t.TaskRunningWait
	}
	return t.DefaultTimeout
}

func (t *manager) TaskHealthCheck() time.Duration {
	if t.TaskHealthCheckWait > 0 {
		return t.TaskHealthCheckWait
	}
	return t.DefaultTimeout
}

func (t *manager) TaskStopped() time.Duration {
	if t.TaskStoppedWait > 0 {
		return t.TaskStoppedWait
	}
	return t.DefaultTimeout
}

func (t *manager) ServiceStable() time.Duration {
	if t.ServiceStableWait > 0 {
		return t.ServiceStableWait
	}
	return t.DefaultTimeout
}
