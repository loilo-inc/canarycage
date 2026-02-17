package commands

import (
	"context"

	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/cli/cage/prompt"
	"github.com/urfave/cli/v3"
)

func (c *CageCommands) Up(input *cageapp.CageCmdInput) *cli.Command {
	return &cli.Command{
		Name:        "up",
		Usage:       "create new ECS service with specified task definition",
		Description: "create new ECS service with specified task definition",
		ArgsUsage:   "[directory path of service.json and task-definition.json]",
		Flags: []cli.Flag{
			cageapp.RegionFlag(&input.Region),
			cageapp.ClusterFlag(&input.Cluster),
			cageapp.ServiceFlag(&input.Service),
			cageapp.TaskDefinitionArnFlag(&input.TaskDefinitionArn),
			cageapp.CanaryTaskIdleDurationFlag(&input.CanaryTaskIdleDuration),
			cageapp.ServiceStableWaitFlag(&input.ServiceStableWait),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir, _, err := RequireArgs(cmd, 1, 1)
			if err != nil {
				return err
			}
			cagecli, err := c.setupCage(input, dir)
			if err != nil {
				return err
			}
			if !input.CI {
				prompter := prompt.NewPrompter(input.Stdin)
				if err := prompter.ConfirmService(input.Envars); err != nil {
					return err
				}
			}
			_, err = cagecli.Up(ctx)
			return err
		},
	}
}
