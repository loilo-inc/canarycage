package cage

import (
	"context"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
)

type Cage interface {
	RollOut(ctx context.Context) *RollOutResult
}

type cage struct {
	cluster           string
	service           string
	region            string
	taskDefinition    *ecs.TaskDefinition
	canaryInstanceArn *string
	ecs               ecsiface.ECSAPI
	alb               elbv2iface.ELBV2API
	ec2               ec2iface.EC2API
}

type Input struct {
	Cluster           string
	Service           string
	Region            string
	CanaryInstanceArn *string
	TaskDefinition    *ecs.TaskDefinition
	ECS               ecsiface.ECSAPI
	ALB               elbv2iface.ELBV2API
	EC2               ec2iface.EC2API
}

func NewCage(input *Input) Cage {
	return &cage{
		cluster:        input.Cluster,
		service:        input.Service,
		region:         input.Region,
		taskDefinition: input.TaskDefinition,
		ecs:            input.ECS,
		alb:            input.ALB,
		ec2:            input.EC2}
}
