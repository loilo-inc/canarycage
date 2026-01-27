package commands

import (
	"context"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/cli/cage/prompt"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/types"
	"github.com/urfave/cli/v2"
)

func (c *CageCommands) RollOut(input *cageapp.CageCmdInput) *cli.Command {
	var updateServiceConf bool
	return &cli.Command{
		Name:        "rollout",
		Usage:       "roll out ECS service to next task definition",
		Description: "start rolling out next service with current service",
		Args:        true,
		ArgsUsage:   "[directory path of service.json and task-definition.json]",
		Flags: []cli.Flag{
			cageapp.RegionFlag(&input.Region),
			cageapp.ClusterFlag(&input.Cluster),
			cageapp.ServiceFlag(&input.Service),
			cageapp.TaskDefinitionArnFlag(&input.TaskDefinitionArn),
			cageapp.CanaryTaskIdleDurationFlag(&input.CanaryTaskIdleDuration),
			&cli.StringFlag{
				Name:        "canaryInstanceArn",
				EnvVars:     []string{env.CanaryInstanceArnKey},
				Usage:       "EC2 instance ARN for placing canary task. required only when LaunchType is EC2",
				Destination: &input.CanaryInstanceArn,
			},
			&cli.BoolFlag{
				Name:        "updateService",
				EnvVars:     []string{env.UpdateServiceKey},
				Usage:       "Update service configurations except for task definiton. Default is false.",
				Destination: &updateServiceConf,
			},
			cageapp.TaskRunningWaitFlag(&input.CanaryTaskRunningWait),
			cageapp.TaskHealthCheckWaitFlag(&input.CanaryTaskHealthCheckWait),
			cageapp.TaskStoppedWaitFlag(&input.CanaryTaskStoppedWait),
			cageapp.ServiceStableWaitFlag(&input.ServiceStableWait),
		},
		Action: func(ctx *cli.Context) error {
			dir, _, err := RequireArgs(ctx, 1, 1)
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
			_, err = cagecli.RollOut(context.Background(), &types.RollOutInput{UpdateService: updateServiceConf})
			return err
		},
	}
}
