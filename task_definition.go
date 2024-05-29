package cage

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"golang.org/x/xerrors"
)

func (c *cage) CreateNextTaskDefinition(ctx context.Context) (*ecstypes.TaskDefinition, error) {
	if c.Env.TaskDefinitionArn != "" {
		log.Infof("--taskDefinitionArn was set to '%s'. skip registering new task definition.", c.Env.TaskDefinitionArn)
		o, err := c.Ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &c.Env.TaskDefinitionArn,
		})
		if err != nil {
			return nil, xerrors.Errorf("failed to describe next task definition: %w", err)
		}
		return o.TaskDefinition, nil
	} else {
		log.Infof("creating next task definition...")
		if out, err := c.Ecs.RegisterTaskDefinition(ctx, c.Env.TaskDefinitionInput); err != nil {
			return nil, xerrors.Errorf("failed to register next task definition: %w", err)
		} else {
			log.Infof(
				"task definition '%s:%d' has been registered",
				*out.TaskDefinition.Family, out.TaskDefinition.Revision,
			)
			return out.TaskDefinition, nil
		}
	}
}
