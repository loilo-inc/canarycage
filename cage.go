package cage

import (
	"context"
	"time"

	"github.com/loilo-inc/canarycage/awsiface"
)

type Cage interface {
	Up(ctx context.Context) (*UpResult, error)
	Run(ctx context.Context, input *RunInput) (*RunResult, error)
	RollOut(ctx context.Context) (*RollOutResult, error)
	Recreate(ctx context.Context) (*RecreateResult, error)
}

type Time interface {
	Now() time.Time
	NewTimer(time.Duration) *time.Timer
}

type cage struct {
	env  *Envars
	ecs  awsiface.EcsClient
	alb  awsiface.AlbClient
	ec2  awsiface.Ec2Client
	time Time
}

type Input struct {
	Env  *Envars
	ECS  awsiface.EcsClient
	ALB  awsiface.AlbClient
	EC2  awsiface.Ec2Client
	Time Time
}

func NewCage(input *Input) Cage {
	if input.Time == nil {
		input.Time = &timeImpl{}
	}
	return &cage{
		env:  input.Env,
		ecs:  input.ECS,
		alb:  input.ALB,
		ec2:  input.EC2,
		time: input.Time,
	}
}
