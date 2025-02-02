package cage

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
)

func (c *cage) CreateNextTaskDefinition(ctx context.Context) (*ecstypes.TaskDefinition, error) {
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	if env.TaskDefinitionArn != "" {
		log.Infof("--taskDefinitionArn was set to '%s'. skip registering new task definition.", env.TaskDefinitionArn)
		o, err := ecsCli.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &env.TaskDefinitionArn,
		})
		if err != nil {
			return nil, err
		}
		return o.TaskDefinition, nil
	} else {
		log.Infof("creating next task definition...")
		if out, err := ecsCli.RegisterTaskDefinition(ctx, env.TaskDefinitionInput); err != nil {
			return nil, err
		} else {
			log.Infof(
				"task definition '%s:%d' has been registered",
				*out.TaskDefinition.Family, out.TaskDefinition.Revision,
			)
			return out.TaskDefinition, nil
		}
	}
}
