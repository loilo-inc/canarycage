package commands

import (
	"context"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/loilo-inc/canarycage"
	"github.com/urfave/cli"
)

func (c *cageCommands) Up() cli.Command {
	envars := cage.Envars{}
	return cli.Command{
		Name: "up",
		Usage: "create new ECS service with specified task definition",
		Description: "create new ECS service with specified task definition",
		ArgsUsage: "[directory path of service.json and task-definition.json (default=.)]",
		Flags: []cli.Flag{
			RegionFlag(&envars.Region),
			ClusterFlag(&envars.Cluster),
			ServiceFlag(&envars.Service),
			TaskDefinitionArnFlag(&envars.TaskDefinitionArn),
		},
		Action: func(ctx *cli.Context) error {
			c.AggregateEnvars(ctx, &envars)
			cagecli := cage.NewCage(&cage.Input{
				Env: &envars,
				ECS: ecs.New(c.ses),
				ALB: elbv2.New(c.ses),
				EC2: ec2.New(c.ses),
			})
			_, err := cagecli.Up(context.Background())
			return err
		},
	}
}

