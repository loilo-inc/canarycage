package commands

import (
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/loilo-inc/canarycage"
	"github.com/urfave/cli"
)

func (c *cageCommands) RollOut() cli.Command {
	var envars = cage.Envars{}
	return cli.Command{
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
			cli.StringFlag{
				Name:        "canaryInstanceArn",
				EnvVar:      cage.CanaryInstanceArnKey,
				Usage:       "EC2 instance ARN for placing canary task. required only when LaunchType is EC2",
				Destination: &envars.CanaryInstanceArn,
			},
		},
		Action: func(ctx *cli.Context) error {
			c.aggregateEnvars(ctx, &envars)
			var ses *session.Session
			if o, err := session.NewSession(&aws.Config{
				Region: &envars.Region,
			}); err != nil {
				return err
			} else {
				ses = o
			}
			cagecli := cage.NewCage(&cage.Input{
				Env: &envars,
				ECS: ecs.New(ses),
				EC2: ec2.New(ses),
				ALB: elbv2.New(ses),
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
