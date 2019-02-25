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
	var region string
	var cluster string
	var service string
	var canaryInstanceArn string
	var taskDefinitionArn string
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
				Usage:       "aws region for ecs. if not specified, try to load from aws sessions automatically",
				Destination: &region,
			},
			cli.StringFlag{
				Name:        "cluster",
				EnvVar:      cage.ClusterKey,
				Usage:       "ecs cluster name. if not specified, load from service.json",
				Destination: &cluster,
			},
			cli.StringFlag{
				Name:        "service",
				EnvVar:      cage.ServiceKey,
				Usage:       "service name. if not specified, load from service.json",
				Destination: &service,
			},
			cli.StringFlag{
				Name:        "canaryInstanceArn",
				EnvVar:      cage.CanaryInstanceArnKey,
				Usage:       "EC2 instance ARN for placing canary task. required only when LaunchType is EC2",
				Destination: &canaryInstanceArn,
			},
			cli.StringFlag{
				Name:        "taskDefinitionArn",
				EnvVar:      cage.TaskDefinitionArnKey,
				Usage:       "full arn for next task definition. if not specified, use task-definition.json for registration",
				Destination: &taskDefinitionArn,
			},
		},
		Action: func(ctx *cli.Context) {
			var _region string
			if region != "" {
				_region = region
				log.Infof("🗺 region was set: %s", _region)
			} else if *c.ses.Config.Region != "" {
				_region = *c.ses.Config.Region
				log.Infof("🗺 region was loaded from sessions: %s", _region)
			} else {
				log.Fatalf("🙄 region must specified by --region flag or aws session")
			}
			envars := &cage.Envars{
				Region:            _region,
				Service:           service,
				Cluster:           cluster,
				TaskDefinitionArn: &taskDefinitionArn,
				CanaryInstanceArn: &canaryInstanceArn,
			}
			if ctx.NArg() > 0 {
				dir := ctx.Args().Get(0)
				td, svc, err := cage.LoadDefinitionsFromFiles(dir);
				if err != nil {
					log.Fatalf(err.Error())
				}
				cage.MergeEnvars(envars, &cage.Envars{
					Cluster:                *svc.Cluster,
					Service:                *svc.ServiceName,
					TaskDefinitionInput:    td,
					ServiceDefinitionInput: svc,
				})
			}
			if err := cage.EnsureEnvars(envars); err != nil {
				log.Fatalf(err.Error())
			}
			if err := Action(c.globalContext, envars, c.ses); err != nil {
				log.Fatalf("failed: %s", err)
			}
		},
	}
}

func Action(
	ctx context.Context,
	envars *cage.Envars,
	ses *session.Session,
) error {
	cagecli := cage.NewCage(&cage.Input{
		Env: envars,
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
