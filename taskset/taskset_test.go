package taskset_test

import (
	"context"
	"fmt"
	"testing"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/golang/mock/gomock"
	"github.com/loilo-inc/canarycage/mocks/mock_task"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/taskset"
	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		factory := mock_task.NewMockFactory(ctrl)
		albTask := mock_task.NewMockTask(ctrl)
		lb := ecstypes.LoadBalancer{}
		gomock.InOrder(
			factory.EXPECT().NewAlbTask(gomock.Any(), &lb).Return(albTask),
			albTask.EXPECT().Start(gomock.Any()).Return(nil),
		)
		albTask.EXPECT().Wait(gomock.Any()).Return(nil)
		albTask.EXPECT().Stop(gomock.Any()).Return(nil)
		input := &taskset.Input{
			Input:         &task.Input{},
			LoadBalancers: []ecstypes.LoadBalancer{lb},
		}
		set := taskset.NewSet(factory, input)
		ctx := context.TODO()
		assert.NoError(t, set.Exec(ctx))
		assert.NoError(t, set.Cleanup(ctx))
	})
	t.Run("should add a simple task if no load balancer or service registry is given", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		factory := mock_task.NewMockFactory(ctrl)
		simpleTask := mock_task.NewMockTask(ctrl)
		input := &taskset.Input{
			Input: &task.Input{},
		}
		factory.EXPECT().NewSimpleTask(input.Input).Return(simpleTask)
		simpleTask.EXPECT().Start(gomock.Any()).Return(nil)
		simpleTask.EXPECT().Wait(gomock.Any()).Return(nil)
		simpleTask.EXPECT().Stop(gomock.Any()).Return(nil)
		set := taskset.NewSet(factory, input)
		ctx := context.TODO()
		assert.NoError(t, set.Exec(ctx))
		assert.NoError(t, set.Cleanup(ctx))
	})
	t.Run("should aggregate errors from task.Wait", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		factory := mock_task.NewMockFactory(ctrl)
		albTask := mock_task.NewMockTask(ctrl)
		lb := ecstypes.LoadBalancer{}
		gomock.InOrder(
			factory.EXPECT().NewAlbTask(gomock.Any(), &lb).Return(albTask),
			albTask.EXPECT().Start(gomock.Any()).Return(nil),
			albTask.EXPECT().Wait(gomock.Any()).Return(fmt.Errorf("error")),
		)
		input := &taskset.Input{
			Input:         &task.Input{},
			LoadBalancers: []ecstypes.LoadBalancer{lb},
		}
		set := taskset.NewSet(factory, input)
		ctx := context.TODO()
		err := set.Exec(ctx)
		assert.EqualError(t, err, "error")
	})

	t.Run("should error immediately if task failed to start", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		factory := mock_task.NewMockFactory(ctrl)
		mtask := mock_task.NewMockTask(ctrl)
		lb := &ecstypes.LoadBalancer{}
		factory.EXPECT().NewAlbTask(gomock.Any(), lb).Return(mtask)
		mtask.EXPECT().Start(gomock.Any()).Return(fmt.Errorf("error"))
		input := &taskset.Input{
			Input:         &task.Input{},
			LoadBalancers: []ecstypes.LoadBalancer{{}},
		}
		set := taskset.NewSet(factory, input)
		ctx := context.TODO()
		err := set.Exec(ctx)
		assert.EqualError(t, err, "error")
	})
}
