package commands

import (
	"context"

	"github.com/apex/log"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/types"
	"github.com/urfave/cli/v2"
)

func (c *CageCommands) RollOut(
	envars *env.Envars,
) *cli.Command {
	var updateServiceConf bool
	return &cli.Command{
		Name:        "rollout",
		Usage:       "roll out ECS service to next task definition",
		Description: "start rolling out next service with current service",
		Args:        true,
		ArgsUsage:   "[directory path of service.json and task-definition.json]",
		Flags: []cli.Flag{
			RegionFlag(&envars.Region),
			ClusterFlag(&envars.Cluster),
			ServiceFlag(&envars.Service),
			TaskDefinitionArnFlag(&envars.TaskDefinitionArn),
			CanaryTaskIdleDurationFlag(&envars.CanaryTaskIdleDuration),
			&cli.StringFlag{
				Name:        "canaryInstanceArn",
				EnvVars:     []string{env.CanaryInstanceArnKey},
				Usage:       "EC2 instance ARN for placing canary task. required only when LaunchType is EC2",
				Destination: &envars.CanaryInstanceArn,
			},
			&cli.BoolFlag{
				Name:        "updateService",
				EnvVars:     []string{env.UpdateServiceKey},
				Usage:       "Update service configurations except for task definiton. Default is false.",
				Destination: &updateServiceConf,
			},
			TaskRunningWaitFlag(&envars.CanaryTaskRunningWait),
			TaskHealthCheckWaitFlag(&envars.CanaryTaskHealthCheckWait),
			TaskStoppedWaitFlag(&envars.CanaryTaskStoppedWait),
			ServiceStableWaitFlag(&envars.ServiceStableWait),
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
			result, err := cagecli.RollOut(context.Background(), &types.RollOutInput{UpdateService: updateServiceConf})
			if err != nil {
				if !result.ServiceUpdated {
					log.Errorf("🤕 failed to roll out new tasks but service '%s' is not changed", envars.Service)
				} else {
					log.Errorf("😭 failed to roll out new tasks and service '%s' might be changed. CHECK ECS CONSOLE NOW!", envars.Service)
				}
				return err
			}
			log.Infof("🎉service roll out has completed successfully!🎉")
			return nil
		},
	}
}
