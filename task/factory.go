package task

import (
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/logos/di"
)

type Factory interface {
	NewAlbTask(input *Input, lb *ecstypes.LoadBalancer) Task
	NewSrvTask(input *Input, srv *ecstypes.ServiceRegistry) Task
	NewSimpleTask(input *Input) Task
}

type factory struct {
	di *di.D
}

func NewFactory(di *di.D) Factory {
	return &factory{di: di}
}

func (f *factory) NewAlbTask(input *Input, lb *ecstypes.LoadBalancer) Task {
	return NewAlbTask(f.di, input, lb)
}

func (f *factory) NewSrvTask(input *Input, srv *ecstypes.ServiceRegistry) Task {
	return NewSrvTask(f.di, input, srv)
}

func (f *factory) NewSimpleTask(input *Input) Task {
	return NewSimpleTask(f.di, input)
}
