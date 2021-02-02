package commands

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
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
			var ses *session.Session
			if o, err := session.NewSession(&aws.Config{
				Region: &envars.Region,
			}); err != nil {
				return err
			} else {
				ses = o
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
				ECS: ecs.New(ses),
				ALB: elbv2.New(ses),
				EC2: ec2.New(ses),
			})
			_, err := cagecli.Run(context.Background(), &cage.RunInput{
				Container: &container,
				Overrides: &ecs.TaskOverride{
					ContainerOverrides: []*ecs.ContainerOverride{
						{Command: aws.StringSlice(commands),
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
