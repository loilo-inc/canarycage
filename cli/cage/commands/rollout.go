package commands

import (
	"encoding/json"
	"fmt"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/loilo-inc/canarycage"
	"github.com/urfave/cli"
	"os"
)

func RollOutCommand() cli.Command {
	envars := &cage.Envars{
		Region:                  aws.String(""),
		Cluster:                 aws.String(""),
		Service:                 aws.String(""),
		ServiceDefinitionBase64: aws.String(""),
		TaskDefinitionBase64:    aws.String(""),
		TaskDefinitionArn:       aws.String(""),
	}
	return cli.Command{
		Name:        "rollout",
		Description: "start rolling out next service with current service",
		ArgsUsage:   "[deploy context path]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "skeleton",
				Usage: "generate config file skeleton json",
			},
			cli.BoolFlag{
				Name:  "dryRun",
				Usage: "describe roll out plan without affecting any resources",
			},
			cli.StringFlag{
				Name:        "region",
				EnvVar:      cage.RegionKey,
				Value:       "us-west-2",
				Usage:       "aws region for ecs",
				Destination: envars.Region,
			},
			cli.StringFlag{
				Name:        "cluster",
				EnvVar:      cage.ClusterKey,
				Usage:       "ecs cluster name",
				Destination: envars.Cluster,
			},
			cli.StringFlag{
				Name:        "service",
				EnvVar:      cage.ServiceKey,
				Usage:       "service name",
				Destination: envars.Service,
			},
			cli.StringFlag{
				Name: "canaryService",
				EnvVar: cage.CanaryServiceKey,
				Usage: "canary service name",
				Destination: envars.CanaryService,
			},
			cli.StringFlag{
				Name:        "serviceDefinitionBase64",
				EnvVar:      cage.ServiceDefinitionBase64Key,
				Usage:       "base64 encoded service definition for next service",
				Destination: envars.ServiceDefinitionBase64,
			},
			cli.StringFlag{
				Name:        "taskDefinitionBase64",
				EnvVar:      cage.TaskDefinitionBase64Key,
				Usage:       "base64 encoded task definition for next task definition",
				Destination: envars.TaskDefinitionBase64,
			},
			cli.StringFlag{
				Name:        "taskDefinitionArn",
				EnvVar:      cage.TaskDefinitionArnKey,
				Usage:       "full arn for next task definition",
				Destination: envars.TaskDefinitionArn,
			},
		},
		Action: func(ctx *cli.Context) {
			if ctx.Bool("skeleton") {
				d, err := json.MarshalIndent(envars, "", "\t")
				if err != nil {
					log.Fatalf("failed to marshal json due to: %s", err)
				}
				fmt.Fprint(os.Stdout, string(d))
				os.Exit(0)
			}
			if ctx.NArg() > 0 {
				// deployコンテクストを指定した場合
				dir := ctx.Args().Get(0)
				if err := envars.LoadFromFiles(dir); err != nil {
					log.Fatalf(err.Error())
				}
			}
			ses, err := session.NewSession(&aws.Config{
				Region: envars.Region,
			})
			if err != nil {
				log.Fatalf("failed to create new AWS session due to: %s", err)
			}
			cageCtx := &cage.Context{
				Ecs: ecs.New(ses),
				Alb: elbv2.New(ses),
			}
			if err := cage.EnsureEnvars(envars); err != nil {
				log.Fatalf(err.Error())
			}
			if err := Action(envars, cageCtx); err != nil {
				log.Fatalf("failed: %s", err)
			}
		},
	}
}

func Action(envars *cage.Envars, ctx *cage.Context) error {
	result := envars.RollOut(ctx)
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
