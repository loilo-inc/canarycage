package commands

import (
	"context"

	cage "github.com/loilo-inc/canarycage"
	"github.com/urfave/cli/v2"
)

func (c *cageCommands) Up(
	envars *cage.Envars,
) *cli.Command {
	return &cli.Command{
		Name:        "up",
		Usage:       "create new ECS service with specified task definition",
		Description: "create new ECS service with specified task definition",
		Args:        true,
		ArgsUsage:   "[directory path of service.json and task-definition.json]",
		Flags: []cli.Flag{
			RegionFlag(&envars.Region),
			ClusterFlag(&envars.Cluster),
			ServiceFlag(&envars.Service),
			TaskDefinitionArnFlag(&envars.TaskDefinitionArn),
			CanaryTaskIdleDurationFlag(&envars.CanaryTaskIdleDuration),
		},
		Action: func(ctx *cli.Context) error {
			dir, _, err := c.requireArgs(ctx, 1, 1)
			if err != nil {
				return err
			}
			cagecli, err := c.setupCage(envars, dir)
			if err != nil {
				return err
			}
			if err := c.Prompt.ConfirmService(envars); err != nil {
				return err
			}
			_, err = cagecli.Up(context.Background())
			return err
		},
	}
}
