package test

import (
	"fmt"
	"sync"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/logos/di"
)

type commons struct {
	Services        map[string]*ecstypes.Service
	Tasks           map[string]*ecstypes.Task
	TaskDefinitions *TaskDefinitionRepository
	TargetGroups    map[string]*TargetGroup
	mux             sync.Mutex
	di              *di.D
}

type MockContext struct {
	*commons
	Ecs awsiface.EcsClient
	Alb awsiface.AlbClient
	Ec2 awsiface.Ec2Client
}

func NewMockContext() *MockContext {
	d := di.NewDomain(func(b *di.B) {
		b.Set(key.Logger, NewLogger())
	})
	cm := &commons{
		Services: make(map[string]*ecstypes.Service),
		Tasks:    make(map[string]*ecstypes.Task),
		TaskDefinitions: &TaskDefinitionRepository{
			families: make(map[string]*TaskDefinitionFamily),
		},
		TargetGroups: make(map[string]*TargetGroup),
		di:           d,
	}
	return &MockContext{
		commons: cm,
		Ecs:     &EcsServer{commons: cm},
		Ec2:     &Ec2Server{commons: cm},
		Alb:     &AlbServer{commons: cm},
	}
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

func (ctx *commons) GetEcsService(id string) (*ecstypes.Service, bool) {
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

var Err = fmt.Errorf("error")
