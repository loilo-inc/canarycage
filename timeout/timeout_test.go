package timeout_test

import (
	"testing"
	"time"

	"github.com/loilo-inc/canarycage/timeout"
	"github.com/stretchr/testify/assert"
)

func TestManager(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		man := timeout.NewManager(10, &timeout.Input{})
		assert.Equal(t, time.Duration(10), man.TaskRunning())
		assert.Equal(t, time.Duration(10), man.TaskStopped())
		assert.Equal(t, time.Duration(10), man.TaskHealthCheck())
		assert.Equal(t, time.Duration(10), man.ServiceStable())
		assert.Equal(t, time.Duration(10), man.TargetHealthCheck())
	})
	t.Run("with config", func(t *testing.T) {
		man := timeout.NewManager(10, &timeout.Input{
			TaskRunningWait:       1,
			TaskStoppedWait:       2,
			TaskHealthCheckWait:   3,
			ServiceStableWait:     4,
			TargetHealthCheckWait: 5,
		})
		assert.Equal(t, time.Duration(1), man.TaskRunning())
		assert.Equal(t, time.Duration(2), man.TaskStopped())
		assert.Equal(t, time.Duration(3), man.TaskHealthCheck())
		assert.Equal(t, time.Duration(4), man.ServiceStable())
		assert.Equal(t, time.Duration(5), man.TargetHealthCheck())
	})
}
