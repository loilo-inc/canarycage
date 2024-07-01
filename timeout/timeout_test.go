package timeout_test

import (
	"testing"
	"time"

	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/timeout"
	"github.com/stretchr/testify/assert"
)

func TestManager(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		man := timeout.NewManager(&env.Envars{}, 10)
		assert.Equal(t, time.Duration(10), man.TaskRunning())
		assert.Equal(t, time.Duration(10), man.TaskStopped())
		assert.Equal(t, time.Duration(10), man.TaskHealthCheck())
		assert.Equal(t, time.Duration(10), man.ServiceStable())
		assert.Equal(t, time.Duration(10), man.TargetHealthCheck())
	})
	t.Run("with config", func(t *testing.T) {
		man := timeout.NewManager(&env.Envars{
			CanaryTaskRunningWait:     1,
			CanaryTaskStoppedWait:     2,
			CanaryTaskHealthCheckWait: 3,
			ServiceStableWait:         4,
			TargetHealthCheckWait:     5,
		}, 10)
		assert.Equal(t, 1*time.Second, man.TaskRunning())
		assert.Equal(t, 2*time.Second, man.TaskStopped())
		assert.Equal(t, 3*time.Second, man.TaskHealthCheck())
		assert.Equal(t, 4*time.Second, man.ServiceStable())
		assert.Equal(t, 5*time.Second, man.TargetHealthCheck())
	})
}
