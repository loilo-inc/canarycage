package taskset

import (
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/task"
)

type Factory interface {
	NewAlbTask(input *task.Input, lb *ecstypes.LoadBalancer) task.Task
	NewSrvTask(input *task.Input, srv *ecstypes.ServiceRegistry) task.Task
	NewSimpleTask(input *task.Input) task.Task
}

type factory struct{}

func NewFactory() Factory {
	return &factory{}
}

func (f *factory) NewAlbTask(input *task.Input, lb *ecstypes.LoadBalancer) task.Task {
	return task.NewAlbTask(input, lb)
}

func (f *factory) NewSrvTask(input *task.Input, srv *ecstypes.ServiceRegistry) task.Task {
	return task.NewSrvTask(input, srv)
}

func (f *factory) NewSimpleTask(input *task.Input) task.Task {
	return task.NewSimpleTask(input)
}
