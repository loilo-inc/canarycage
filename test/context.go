package test

import (
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
)

type commons struct {
	Services        map[string]*types.Service
	Tasks           map[string]*types.Task
	TaskDefinitions *TaskDefinitionRepository
	TargetGroups    map[string]struct{}
	mux             sync.Mutex
}

type MockContext struct {
	*commons
	awsiface.EcsClient
	awsiface.AlbClient
	awsiface.Ec2Client
	awsiface.SrvClient
}

func NewMockContext() *MockContext {
	cm := &commons{
		Services: make(map[string]*types.Service),
		Tasks:    make(map[string]*types.Task),
		TaskDefinitions: &TaskDefinitionRepository{
			families: make(map[string]*TaskDefinitionFamily),
		},
		TargetGroups: make(map[string]struct{}),
	}
	return &MockContext{
		commons:   cm,
		EcsClient: &EcsServer{commons: cm},
		Ec2Client: &Ec2Server{commons: cm},
		SrvClient: &SrvServer{commons: cm},
		AlbClient: &AlbServer{commons: cm},
	}
}

func (ctx *commons) GetTask(id string) (*types.Task, bool) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	o, ok := ctx.Tasks[id]
	return o, ok
}

func (ctx *commons) RunningTaskSize() int {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()

	count := 0
	for _, v := range ctx.Tasks {
		if v.LastStatus != nil && *v.LastStatus == "RUNNING" {
			count++
		}
	}

	return count
}

func (ctx *commons) GetEcsService(id string) (*types.Service, bool) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	o, ok := ctx.Services[id]
	return o, ok
}

func (ctx *commons) ActiveServiceSize() (count int) {
	ctx.mux.Lock()
	defer ctx.mux.Unlock()
	for _, v := range ctx.Services {
		if v.Status != nil && *v.Status == "ACTIVE" {
			count++
		}
	}
	return
}
