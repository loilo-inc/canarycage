package env

import (
	"time"
)

const defaultTimeout = 15 * time.Minute

func (t *Envars) TaskRunning() time.Duration {
	wait := t.CanaryTaskRunningWait
	if wait > 0 {
		return time.Duration(wait) * time.Second
	}
	return defaultTimeout
}

func (t *Envars) TaskHealthCheck() time.Duration {
	wait := t.CanaryTaskHealthCheckWait
	if wait > 0 {
		return time.Duration(wait) * time.Second
	}
	return defaultTimeout
}

func (t *Envars) TaskStopped() time.Duration {
	wait := t.CanaryTaskStoppedWait
	if wait > 0 {
		return time.Duration(wait) * time.Second
	}
	return defaultTimeout
}

func (t *Envars) ServiceStable() time.Duration {
	wait := t.ServiceStableWait
	if wait > 0 {
		return time.Duration(wait) * time.Second
	}
	return defaultTimeout
}
