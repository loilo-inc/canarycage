package cage

import (
	"context"

	"github.com/loilo-inc/canarycage/awsiface"
)

type Cage interface {
	Up(ctx context.Context) (*UpResult, error)
	Run(ctx context.Context, input *RunInput) (*RunResult, error)
	RollOut(ctx context.Context) (*RollOutResult, error)
}

type cage struct {
	env *Envars
	ecs awsiface.EcsClient
	alb awsiface.AlbClient
	ec2 awsiface.Ec2Client
}

type Input struct {
	Env *Envars
	ECS awsiface.EcsClient
	ALB awsiface.AlbClient
	EC2 awsiface.Ec2Client
}

func NewCage(input *Input) Cage {
	return &cage{
		env: input.Env,
		ecs: input.ECS,
		alb: input.ALB,
		ec2: input.EC2,
	}
}
