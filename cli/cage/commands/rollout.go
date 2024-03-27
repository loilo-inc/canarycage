package commands

import (
	"context"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/loilo-inc/canarycage"
	"github.com/urfave/cli/v2"
)

func (c *cageCommands) RollOut() *cli.Command {
	var envars = cage.Envars{}
	return &cli.Command{
		Name:        "rollout",
		Usage:       "roll out ECS service to next task definition",
		Description: "start rolling out next service with current service",
		ArgsUsage:   "[directory path of service.json and task-definition.json (default=.)]",
		Flags: []cli.Flag{
			RegionFlag(&envars.Region),
			ClusterFlag(&envars.Cluster),
			ServiceFlag(&envars.Service),
			TaskDefinitionArnFlag(&envars.TaskDefinitionArn),
			CanaryTaskIdleDurationFlag(&envars.CanaryTaskIdleDuration),
			&cli.StringFlag{
				Name:        "canaryInstanceArn",
				EnvVars:     []string{cage.CanaryInstanceArnKey},
				Usage:       "EC2 instance ARN for placing canary task. required only when LaunchType is EC2",
				Destination: &envars.CanaryInstanceArn,
			},
		},
		Action: func(ctx *cli.Context) error {
			c.aggregateEnvars(ctx, &envars)
			var cfg aws.Config
			if o, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(envars.Region)); err != nil {
				return err
			} else {
				cfg = o
			}
			cagecli := cage.NewCage(&cage.Input{
				Env: &envars,
				ECS: ecs.NewFromConfig(cfg),
				EC2: ec2.NewFromConfig(cfg),
				ALB: elbv2.NewFromConfig(cfg),
			})
			result, err := cagecli.RollOut(c.ctx)
			if err != nil {
				log.Error(err.Error())
				if result.ServiceIntact {
					log.Errorf("🤕 failed to roll out new tasks but service '%s' is not changed", envars.Service)
				} else {
					log.Errorf("😭 failed to roll out new tasks and service '%s' might be changed. check in console!!", envars.Service)
				}
				return err
			}
			log.Infof("🎉service roll out has completed successfully!🎉")
			return nil
		},
	}
}
