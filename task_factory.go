package cage

import (
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/task"
)

type TaskFactory interface {
	NewAlbTask(input *task.Input, lb *ecstypes.LoadBalancer) task.Task
	NewSrvTask(input *task.Input, srv *ecstypes.ServiceRegistry) task.Task
	NewSimpleTask(input *task.Input) task.Task
}

type taskFactory struct{}

func (f *taskFactory) NewAlbTask(input *task.Input, lb *ecstypes.LoadBalancer) task.Task {
	return task.NewAlbTask(input, lb)
}

func (f *taskFactory) NewSrvTask(input *task.Input, srv *ecstypes.ServiceRegistry) task.Task {
	return task.NewSrvTask(input, srv)
}

func (f *taskFactory) NewSimpleTask(input *task.Input) task.Task {
	return task.NewSimpleTask(input)
}
