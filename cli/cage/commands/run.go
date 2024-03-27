package commands

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	cage "github.com/loilo-inc/canarycage"
	"github.com/urfave/cli/v2"
)

func (c *cageCommands) Run() *cli.Command {
	envars := cage.Envars{}
	return &cli.Command{
		Name:        "run",
		Usage:       "run task with specified task definition",
		Description: "run task with specified task definition",
		ArgsUsage:   "<directory path of service.json and task-definition.json> <container> <commands>...",
		Flags: []cli.Flag{
			RegionFlag(&envars.Region),
			ClusterFlag(&envars.Cluster),
		},
		Action: func(ctx *cli.Context) error {
			c.aggregateEnvars(ctx, &envars)
			var cfg aws.Config
			if o, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(envars.Region)); err != nil {
				return err
			} else {
				cfg = o
			}
			rest := ctx.Args().Tail()
			if len(rest) < 1 {
				log.Error("<container> required")
				return fmt.Errorf("")
			}
			if len(rest) < 2 {
				log.Errorf("<commands> required")
				return fmt.Errorf("")
			}
			container := rest[0]
			commands := rest[1:]
			cagecli := cage.NewCage(&cage.Input{
				Env: &envars,
				ECS: ecs.NewFromConfig(cfg),
				EC2: ec2.NewFromConfig(cfg),
				ALB: elbv2.NewFromConfig(cfg),
			})
			_, err := cagecli.Run(context.Background(), &cage.RunInput{
				Container: &container,
				Overrides: &types.TaskOverride{
					ContainerOverrides: []types.ContainerOverride{
						{Command: commands,
							Name: &container},
					},
				},
			})
			if err != nil {
				log.Error(err.Error())
				return err
			}
			log.Infof("👍 task successfully executed")
			return nil
		},
	}
}
