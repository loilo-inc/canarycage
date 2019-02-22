package commands

import (
	"context"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/loilo-inc/canarycage"
	"github.com/urfave/cli"
)

func (c *cageCommands) RollOut() cli.Command {
	envarsFromCli := cage.Envars{}
	return cli.Command{
		Name:        "rollout",
		Description: "start rolling out next service with current service",
		ArgsUsage:   "[deploy context path]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "dryRun",
				Usage: "describe roll out plan without affecting any resources",
			},
			cli.StringFlag{
				Name:        "region",
				EnvVar:      cage.RegionKey,
				Value:       "us-west-2",
				Usage:       "aws region for ecs",
				Destination: &envarsFromCli.Region,
			},
			cli.StringFlag{
				Name:        "cluster",
				EnvVar:      cage.ClusterKey,
				Usage:       "ecs cluster name",
				Destination: &envarsFromCli.Cluster,
			},
			cli.StringFlag{
				Name:        "service",
				EnvVar:      cage.ServiceKey,
				Usage:       "service name",
				Destination: &envarsFromCli.Service,
			},
			cli.StringFlag{
				Name:        "canaryInstanceArn",
				EnvVar:      cage.CanaryInstanceArnKey,
				Usage:       "canary instance ARN (required only EC2 ECS)",
				Destination: &envarsFromCli.CanaryInstanceArn,
			},
			cli.StringFlag{
				Name:        "taskDefinitionArn",
				EnvVar:      cage.TaskDefinitionArnKey,
				Usage:       "full arn for next task definition",
				Destination: &envarsFromCli.TaskDefinitionArn,
			},
		},
		Action: func(ctx *cli.Context) {
			envars := envarsFromCli
			if ctx.NArg() > 0 {
				// deployコンテクストを指定した場合
				dir := ctx.Args().Get(0)
				var envarsFromFiles *cage.Envars
				if d, err := cage.LoadEnvarsFromFiles(dir); err != nil {
					log.Fatalf(err.Error())
				} else {
					envarsFromFiles = d
				}
				cage.MergeEnvars(&envars, envarsFromFiles)
			}
			if err := cage.EnsureEnvars(&envars); err != nil {
				log.Fatalf(err.Error())
			}
			if err := Action(c.globalContext, &envars, c.ses); err != nil {
				log.Fatalf("failed: %s", err)
			}
		},
	}
}

func Action(
	ctx context.Context,
	envars *cage.Envars,
	ses *session.Session) error {
	cagecli := cage.NewCage(&cage.Input{
		Env:envars,
		ECS: ecs.New(ses),
		EC2: ec2.New(ses),
		ALB: elbv2.New(ses),
	})
	result := cagecli.RollOut(ctx)
	if result.Error != nil {
		if result.ServiceIntact {
			log.Errorf("🤕 failed to roll out new tasks but service '%s' is not changed. error: %s", result.Error)
		} else {
			log.Errorf("😭 failed to roll out new tasks and service '%s' might be changed. check in console!!. error: %s", result.Error)
		}
		return result.Error
	}
	log.Infof("🎉service roll out has completed successfully!🎉")
	return nil
}
