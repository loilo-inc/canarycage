package commands

import (
	"context"

	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/cli/cage/prompt"
	"github.com/loilo-inc/canarycage/v5/env"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/urfave/cli/v3"
)

func (c *CageCommands) RollOut(input *cageapp.CageCmdInput) *cli.Command {
	var updateServiceConf bool
	return &cli.Command{
		Name:        "rollout",
		Usage:       "roll out ECS service to next task definition",
		Description: "start rolling out next service with current service",
		ArgsUsage:   "[directory path of service.json and task-definition.json]",
		Flags: []cli.Flag{
			cageapp.RegionFlag(&input.Region),
			cageapp.ClusterFlag(&input.Cluster),
			cageapp.ServiceFlag(&input.Service),
			cageapp.TaskDefinitionArnFlag(&input.TaskDefinitionArn),
			cageapp.CanaryTaskIdleDurationFlag(&input.CanaryTaskIdleDuration),
			&cli.StringFlag{
				Name:        "canaryInstanceArn",
				Sources:     cli.EnvVars(env.CanaryInstanceArnKey),
				Usage:       "EC2 instance ARN for placing canary task. required only when LaunchType is EC2",
				Destination: &input.CanaryInstanceArn,
			},
			&cli.BoolFlag{
				Name:        "updateService",
				Sources:     cli.EnvVars(env.UpdateServiceKey),
				Usage:       "Update service configurations except for task definiton. Default is false.",
				Destination: &updateServiceConf,
			},
			cageapp.TaskRunningWaitFlag(&input.CanaryTaskRunningWait),
			cageapp.TaskHealthCheckWaitFlag(&input.CanaryTaskHealthCheckWait),
			cageapp.TaskStoppedWaitFlag(&input.CanaryTaskStoppedWait),
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
			_, err = cagecli.RollOut(ctx, &types.RollOutInput{UpdateService: updateServiceConf})
			return err
		},
	}
}
