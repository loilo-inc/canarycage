//go:generate go run github.com/golang/mock/mockgen -source $GOFILE -destination ../mocks/mock_$GOPACKAGE/$GOFILE -package mock_$GOPACKAGE
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
	Env     *Envars
	Ecs     awsiface.EcsClient
	Alb     awsiface.AlbClient
	Ec2     awsiface.Ec2Client
	Time    Time
	MaxWait time.Duration
}

type Input struct {
	Env     *Envars
	ECS     awsiface.EcsClient
	ALB     awsiface.AlbClient
	EC2     awsiface.Ec2Client
	Time    Time
	MaxWait time.Duration
}

func NewCage(input *Input) Cage {
	if input.Time == nil {
		input.Time = &timeImpl{}
	}
	return &cage{
		Env:     input.Env,
		Ecs:     input.ECS,
		Alb:     input.ALB,
		Ec2:     input.EC2,
		Time:    input.Time,
		MaxWait: 5 * time.Minute,
	}
}
