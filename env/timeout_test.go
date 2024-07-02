package env_test

import (
	"testing"
	"time"

	"github.com/loilo-inc/canarycage/env"
	"github.com/stretchr/testify/assert"
)

func TestEnv_Timeout(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		e := &env.Envars{}
		assert.Equal(t, 15*time.Minute, e.GetTaskRunningWait())
		assert.Equal(t, 15*time.Minute, e.GetTaskStoppedWait())
		assert.Equal(t, 15*time.Minute, e.GetTaskHealthCheckWait())
		assert.Equal(t, 15*time.Minute, e.GetServiceStableWait())
		assert.Equal(t, time.Duration(0), e.GetCanaryTaskIdleWait())
	})
	t.Run("with config", func(t *testing.T) {
		e := &env.Envars{
			CanaryTaskRunningWait:     1,
			CanaryTaskStoppedWait:     2,
			CanaryTaskHealthCheckWait: 3,
			ServiceStableWait:         4,
			CanaryTaskIdleDuration:    5,
		}
		assert.Equal(t, 1*time.Second, e.GetTaskRunningWait())
		assert.Equal(t, 2*time.Second, e.GetTaskStoppedWait())
		assert.Equal(t, 3*time.Second, e.GetTaskHealthCheckWait())
		assert.Equal(t, 4*time.Second, e.GetServiceStableWait())
		assert.Equal(t, 5*time.Second, e.GetCanaryTaskIdleWait())
	})
}
