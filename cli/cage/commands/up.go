package commands

import (
	"context"
	"os"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/cli/cage/prompt"
	"github.com/loilo-inc/canarycage/env"
	"github.com/urfave/cli/v2"
)

func (c *CageCommands) Up(flag *cageapp.Flag) *cli.Command {
	envars := &env.Envars{}
	return &cli.Command{
		Name:        "up",
		Usage:       "create new ECS service with specified task definition",
		Description: "create new ECS service with specified task definition",
		Args:        true,
		ArgsUsage:   "[directory path of service.json and task-definition.json]",
		Flags: []cli.Flag{
			cageapp.RegionFlag(&envars.Region),
			cageapp.ClusterFlag(&envars.Cluster),
			cageapp.ServiceFlag(&envars.Service),
			cageapp.TaskDefinitionArnFlag(&envars.TaskDefinitionArn),
			cageapp.CanaryTaskIdleDurationFlag(&envars.CanaryTaskIdleDuration),
			cageapp.ServiceStableWaitFlag(&envars.ServiceStableWait),
		},
		Action: func(ctx *cli.Context) error {
			dir, _, err := RequireArgs(ctx, 1, 1)
			if err != nil {
				return err
			}
			cagecli, err := c.setupCage(envars, dir)
			if err != nil {
				return err
			}
			if !flag.CI {
				prompter := prompt.NewPrompter(os.Stdin)
				if err := prompter.ConfirmService(envars); err != nil {
					return err
				}
			}
			_, err = cagecli.Up(context.Background())
			return err
		},
	}
}
