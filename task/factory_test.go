package task

import (
	"testing"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/logos/di"
	"github.com/stretchr/testify/assert"
)

func TestFactory(t *testing.T) {
	d := &di.D{}
	t.Run("NewAlbTask", func(t *testing.T) {
		f := NewFactory(d)
		input := &Input{}
		lb := &ecstypes.LoadBalancer{}
		task := f.NewAlbTask(input, lb)
		v, ok := task.(*albTask)
		assert.NotNil(t, task)
		assert.True(t, ok)
		assert.Equal(t, input, v.Input)
		assert.Equal(t, lb, v.lb)
	})
	t.Run("NewSimpleTask", func(t *testing.T) {
		f := NewFactory(d)
		input := &Input{}
		task := f.NewSimpleTask(input)
		v, ok := task.(*simpleTask)
		assert.NotNil(t, task)
		assert.True(t, ok)
		assert.Equal(t, input, v.Input)
		assert.Equal(t, d, v.di)
	})
}
