package commands

import (
	"context"

	"github.com/apex/log"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/cli/cage/prompt"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/types"
	"github.com/urfave/cli/v2"
)

func (c *CageCommands) Run(app *cageapp.App) *cli.Command {
	envars := &env.Envars{}
	return &cli.Command{
		Name:        "run",
		Usage:       "run task with specified task definition",
		Description: "run task with specified task definition",
		Args:        true,
		ArgsUsage:   "<directory path of service.json and task-definition.json> <container> <commands>...",
		Flags: []cli.Flag{
			cageapp.RegionFlag(&envars.Region),
			cageapp.ClusterFlag(&envars.Cluster),
			cageapp.TaskRunningWaitFlag(&envars.CanaryTaskRunningWait),
			cageapp.TaskStoppedWaitFlag(&envars.CanaryTaskStoppedWait),
		},
		Action: func(ctx *cli.Context) error {
			dir, rest, err := RequireArgs(ctx, 3, 100)
			if err != nil {
				return err
			}
			cagecli, err := c.setupCage(envars, dir)
			if err != nil {
				return err
			}
			if !app.CI {
				prompter := prompt.NewPrompter(app.Stdin)
				if err := prompter.ConfirmTask(envars); err != nil {
					return err
				}
			}
			container := rest[0]
			commands := rest[1:]
			if _, err := cagecli.Run(context.Background(), &types.RunInput{
				Container: &container,
				Overrides: &ecstypes.TaskOverride{
					ContainerOverrides: []ecstypes.ContainerOverride{
						{Command: commands,
							Name: &container},
					},
				},
			}); err != nil {
				return err
			}
			log.Infof("👍 task successfully executed")
			return nil
		},
	}
}
