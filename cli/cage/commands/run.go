package commands

import (
	"context"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/cli/cage/prompt"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/urfave/cli/v3"
)

func (c *CageCommands) Run(input *cageapp.CageCmdInput) *cli.Command {
	return &cli.Command{
		Name:        "run",
		Usage:       "run task with specified task definition",
		Description: "run task with specified task definition",
		ArgsUsage:   "<directory path of service.json and task-definition.json> <container> <commands>...",
		Flags: []cli.Flag{
			cageapp.RegionFlag(&input.Region),
			cageapp.ClusterFlag(&input.Cluster),
			cageapp.TaskRunningWaitFlag(&input.CanaryTaskRunningWait),
			cageapp.TaskStoppedWaitFlag(&input.CanaryTaskStoppedWait),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir, rest, err := RequireArgs(cmd, 3, 100)
			if err != nil {
				return err
			}
			cagecli, err := c.setupCage(input, dir)
			if err != nil {
				return err
			}
			if !input.CI {
				prompter := prompt.NewPrompter(input.Stdin)
				if err := prompter.ConfirmTask(input.Envars); err != nil {
					return err
				}
			}
			container := rest[0]
			commands := rest[1:]
			_, err = cagecli.Run(ctx, &types.RunInput{
				Container: &container,
				Overrides: &ecstypes.TaskOverride{
					ContainerOverrides: []ecstypes.ContainerOverride{
						{Command: commands,
							Name: &container},
					},
				},
			})
			return err
		},
	}
}
