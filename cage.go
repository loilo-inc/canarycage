package cage

import (
	"context"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
)

type Cage interface {
	Up(ctx context.Context) (*UpResult, error)
	Run(ctx context.Context, input *RunInput) (*RunResult, error)
	RollOut(ctx context.Context) (*RollOutResult, error)
}

type cage struct {
	env *Envars
	ecs ecsiface.ECSAPI
	alb elbv2iface.ELBV2API
	ec2 ec2iface.EC2API
}

type Input struct {
	Env *Envars
	ECS ecsiface.ECSAPI
	ALB elbv2iface.ELBV2API
	EC2 ec2iface.EC2API
}

func NewCage(input *Input) Cage {
	return &cage{
		env: input.Env,
		ecs: input.ECS,
		alb: input.ALB,
		ec2: input.EC2,
	}
}
