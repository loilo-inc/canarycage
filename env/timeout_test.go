package env_test

import (
	"testing"
	"time"

	"github.com/loilo-inc/canarycage/env"
	"github.com/stretchr/testify/assert"
)

func TestManager(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		man := &env.Envars{}
		assert.Equal(t, 15*time.Minute, man.TaskRunning())
		assert.Equal(t, 15*time.Minute, man.TaskStopped())
		assert.Equal(t, 15*time.Minute, man.TaskHealthCheck())
		assert.Equal(t, 15*time.Minute, man.ServiceStable())
	})
	t.Run("with config", func(t *testing.T) {
		man := &env.Envars{
			CanaryTaskRunningWait:     1,
			CanaryTaskStoppedWait:     2,
			CanaryTaskHealthCheckWait: 3,
			ServiceStableWait:         4,
		}
		assert.Equal(t, 1*time.Second, man.TaskRunning())
		assert.Equal(t, 2*time.Second, man.TaskStopped())
		assert.Equal(t, 3*time.Second, man.TaskHealthCheck())
		assert.Equal(t, 4*time.Second, man.ServiceStable())
	})
}
