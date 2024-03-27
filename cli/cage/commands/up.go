package commands

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/loilo-inc/canarycage"
	"github.com/urfave/cli/v2"
)

func (c *cageCommands) Up() *cli.Command {
	envars := cage.Envars{}
	return &cli.Command{
		Name:        "up",
		Usage:       "create new ECS service with specified task definition",
		Description: "create new ECS service with specified task definition",
		ArgsUsage:   "[directory path of service.json and task-definition.json (default=.)]",
		Flags: []cli.Flag{
			RegionFlag(&envars.Region),
			ClusterFlag(&envars.Cluster),
			ServiceFlag(&envars.Service),
			TaskDefinitionArnFlag(&envars.TaskDefinitionArn),
			CanaryTaskIdleDurationFlag(&envars.CanaryTaskIdleDuration),
		},
		Action: func(ctx *cli.Context) error {
			c.aggregateEnvars(ctx, &envars)
			var cfg aws.Config
			if o, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(envars.Region)); err != nil {
				return err
			} else {
				cfg = o
			}
			cagecli := cage.NewCage(&cage.Input{
				Env: &envars,
				ECS: ecs.NewFromConfig(cfg),
				EC2: ec2.NewFromConfig(cfg),
				ALB: elbv2.NewFromConfig(cfg),
			})
			_, err := cagecli.Up(context.Background())
			if err != nil {
				log.Error(err.Error())
			}
			return err
		},
	}
}
