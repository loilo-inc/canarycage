package taskset_test

import (
	"testing"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/taskset"
	"github.com/stretchr/testify/assert"
)

func TestFactory(t *testing.T) {
	t.Run("NewAlbTask", func(t *testing.T) {
		f := taskset.NewFactory()
		input := &task.Input{}
		lb := &ecstypes.LoadBalancer{}
		task := f.NewAlbTask(input, lb)
		assert.NotNil(t, task)
	})
	t.Run("NewSrvTask", func(t *testing.T) {
		f := taskset.NewFactory()
		input := &task.Input{}
		srv := &ecstypes.ServiceRegistry{}
		task := f.NewSrvTask(input, srv)
		assert.NotNil(t, task)
	})
	t.Run("NewSimpleTask", func(t *testing.T) {
		f := taskset.NewFactory()
		input := &task.Input{}
		task := f.NewSimpleTask(input)
		assert.NotNil(t, task)
	})
}
