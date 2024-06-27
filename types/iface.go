package types

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
)

type Envars struct {
	_                         struct{} `type:"struct"`
	CI                        bool     `json:"ci" type:"bool"`
	Region                    string   `json:"region" type:"string"`
	Cluster                   string   `json:"cluster" type:"string" required:"true"`
	Service                   string   `json:"service" type:"string" required:"true"`
	CanaryInstanceArn         string
	TaskDefinitionArn         string `json:"nextTaskDefinitionArn" type:"string"`
	TaskDefinitionInput       *ecs.RegisterTaskDefinitionInput
	ServiceDefinitionInput    *ecs.CreateServiceInput
	CanaryTaskIdleDuration    int // sec
	CanaryTaskRunningWait     int // sec
	CanaryTaskHealthCheckWait int // sec
	CanaryTaskStoppedWait     int // sec
	ServiceStableWait         int // sec
}

type Cage interface {
	Up(ctx context.Context) (*UpResult, error)
	Run(ctx context.Context, input *RunInput) (*RunResult, error)
	RollOut(ctx context.Context, input *RollOutInput) (*RollOutResult, error)
}

type Time interface {
	Now() time.Time
	NewTimer(time.Duration) *time.Timer
}

type Input struct {
	Env  *Envars
	Ecs  awsiface.EcsClient
	Alb  awsiface.AlbClient
	Ec2  awsiface.Ec2Client
	Srv  awsiface.SrvClient
	Time Time
}
type RunInput struct {
	Container *string
	Overrides *ecstypes.TaskOverride
}

type RunResult struct {
	ExitCode int32
}

type RollOutInput struct {
	// UpdateService is a flag to update service with changed configurations except for task definition
	UpdateService bool
}

type RollOutResult struct {
	ServiceIntact bool
}

type UpResult struct {
	TaskDefinition *ecstypes.TaskDefinition
	Service        *ecstypes.Service
}
