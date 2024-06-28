package test

import (
	"sync"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	srvtypes "github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/loilo-inc/canarycage/awsiface"
)

type commons struct {
	Services        map[string]*ecstypes.Service
	Tasks           map[string]*ecstypes.Task
	TaskDefinitions *TaskDefinitionRepository
	TargetGroups    map[string]struct{}
	SrvNamespaces   []*srvtypes.Namespace
	SrvServices     []*srvtypes.Service
	// service.Name -> []*instance
	SrvInsts map[string][]*srvtypes.Instance
	// instance.Id -> HealthStatus
	SrvInstHelths map[string]srvtypes.HealthStatus
	mux           sync.Mutex
}

type MockContext struct {
	*commons
	Ecs awsiface.EcsClient
	Alb awsiface.AlbClient
	Ec2 awsiface.Ec2Client
	Srv awsiface.SrvClient
}

func NewMockContext() *MockContext {
	cm := &commons{
		Services: make(map[string]*ecstypes.Service),
		Tasks:    make(map[string]*ecstypes.Task),
		TaskDefinitions: &TaskDefinitionRepository{
			families: make(map[string]*TaskDefinitionFamily),
		},
		TargetGroups:  make(map[string]struct{}),
		SrvServices:   make([]*srvtypes.Service, 0),
		SrvInsts:      make(map[string][]*srvtypes.Instance),
		SrvInstHelths: make(map[string]srvtypes.HealthStatus),
	}
	return &MockContext{
		commons: cm,
		Ecs:     &EcsServer{commons: cm},
		Ec2:     &Ec2Server{commons: cm},
		Srv:     &SrvServer{commons: cm},
		Alb:     &AlbServer{commons: cm},
	}
}

func (ctx *commons) GetTask(id string) (*ecstypes.Task, bool) {
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
