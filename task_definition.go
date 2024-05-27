package cage

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func (c *cage) CreateNextTaskDefinition(ctx context.Context) (*ecstypes.TaskDefinition, error) {
	if c.env.TaskDefinitionArn != "" {
		log.Infof("--taskDefinitionArn was set to '%s'. skip registering new task definition.", c.env.TaskDefinitionArn)
		o, err := c.ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &c.env.TaskDefinitionArn,
		})
		if err != nil {
			log.Errorf(
				"failed to describe next task definition '%s' due to: %s",
				c.env.TaskDefinitionArn, err,
			)
			return nil, err
		}
		return o.TaskDefinition, nil
	} else {
		log.Infof("creating next task definition...")
		if out, err := c.ecs.RegisterTaskDefinition(ctx, c.env.TaskDefinitionInput); err != nil {
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
