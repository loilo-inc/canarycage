package commands

import (
	"context"

	"github.com/apex/log"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	cage "github.com/loilo-inc/canarycage"
	"github.com/urfave/cli/v2"
)

func (c *cageCommands) Run(
	envars *cage.Envars,
) *cli.Command {
	return &cli.Command{
		Name:        "run",
		Usage:       "run task with specified task definition",
		Description: "run task with specified task definition",
		Args:        true,
		ArgsUsage:   "<directory path of service.json and task-definition.json> <container> <commands>...",
		Flags: []cli.Flag{
			RegionFlag(&envars.Region),
			ClusterFlag(&envars.Cluster),
		},
		Action: func(ctx *cli.Context) error {
			dir, rest, err := c.requireArgs(ctx, 3, 100)
			if err != nil {
				return err
			}
			cagecli, err := c.setupCage(envars, dir)
			if err != nil {
				return err
			}
			if err := c.Prompt.ConfirmTask(envars); err != nil {
				return err
			}
			container := rest[0]
			commands := rest[1:]
			if _, err := cagecli.Run(context.Background(), &cage.RunInput{
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
