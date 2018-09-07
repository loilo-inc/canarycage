package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/apex/log"
	"github.com/urfave/cli"
	"os"
	"github.com/loilo-inc/canarycage"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

func main() {
	// cliのdestinationがnil pointerに代入してくれないので無効値を入れておく
	envars := &cage.Envars{
		Region:                      aws.String(""),
		Cluster:                     aws.String(""),
		NextServiceName:             aws.String(""),
		NextServiceDefinitionBase64: aws.String(""),
		CurrentServiceName:          aws.String(""),
		NextTaskDefinitionBase64:    aws.String(""),
		NextTaskDefinitionArn:       aws.String(""),
		AvailabilityThreshold:       aws.Float64(-1.0),
		ResponseTimeThreshold:       aws.Float64(-1.0),
		RollOutPeriod:               aws.Int64(-1),
		SkipCanary:                  aws.Bool(false),
	}
	configPath := ""
	app := cli.NewApp()
	app.Name = "canarycage"
	app.Version = "1.1.0-alpha"
	app.Description = "A gradual roll-out deployment tool for AWS ECS"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config, c",
			EnvVar:      cage.ConfigKey,
			Usage:       "config file path",
			Destination: &configPath,
		},
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
			Name:        "nextServiceName",
			EnvVar:      cage.NextServiceNameKey,
			Usage:       "next service name",
			Destination: envars.NextServiceName,
		},
		cli.StringFlag{
			Name:        "currentServiceName",
			EnvVar:      cage.CurrentServiceNameKey,
			Usage:       "current service name",
			Destination: envars.CurrentServiceName,
		},
		cli.StringFlag{
			Name:        "nextServiceDefinitionBase64",
			EnvVar:      cage.NextServiceDefinitionBase64Key,
			Usage:       "base64 encoded service definition for next service",
			Destination: envars.NextServiceDefinitionBase64,
		},
		cli.StringFlag{
			Name:        "nextTaskDefinitionBase64",
			EnvVar:      cage.NextTaskDefinitionBase64Key,
			Usage:       "base64 encoded task definition for next task definition",
			Destination: envars.NextTaskDefinitionBase64,
		},
		cli.StringFlag{
			Name:        "nextTaskDefinitionArn",
			EnvVar:      cage.NextTaskDefinitionArnKey,
			Usage:       "full arn for next task definition",
			Destination: envars.NextTaskDefinitionArn,
		},
		cli.Float64Flag{
			Name:        "availabilityThreshold",
			EnvVar:      cage.AvailabilityThresholdKey,
			Usage:       "availability (request success rate) threshold used to evaluate service health by CloudWatch",
			Value:       0.9970,
			Destination: envars.AvailabilityThreshold,
		},
		cli.Float64Flag{
			Name:        "responseTimeThreshold",
			EnvVar:      cage.ResponseTimeThresholdKey,
			Usage:       "average response time (sec) threshold used to evaluate service health by CloudWatch",
			Value:       1.0,
			Destination: envars.ResponseTimeThreshold,
		},
		cli.Int64Flag{
			Name:        "rollOutPeriod",
			EnvVar:      cage.RollOutPeriodKey,
			Usage:       "each roll out period (sec)",
			Value:       300,
			Destination: envars.RollOutPeriod,
		},
		cli.BoolFlag{
			Name:        "skipCanary",
			EnvVar:      cage.SkipCanaryKey,
			Usage:       "skip canary test. ensuring only healthy tasks.",
			Destination: envars.SkipCanary,
		},
	}
	app.Action = func(ctx *cli.Context) {
		if ctx.Bool("skeleton") {
			d, err := json.MarshalIndent(envars, "", "\t")
			if err != nil {
				log.Fatalf("failed to marshal json due to: %s", err)
			}
			fmt.Fprint(os.Stdout, string(d))
			os.Exit(0)
		}
		if configPath != "" {
			d, err := ioutil.ReadFile(configPath)
			if err != nil {
				log.Fatalf("failed to read config file %s due to: %s", configPath, err)
			}
			if err := json.Unmarshal(d, envars); err != nil {
				log.Fatalf("failed to unmarshal json due to: %s", err)
			}
		}
		err := cage.EnsureEnvars(envars)
		if err != nil {
			log.Fatalf(err.Error())
		}
		if ctx.Bool("dryRun") {
			DryRun(envars)
		} else {
			if err := Action(envars); err != nil {
				log.Fatalf("failed: %s", err)
			}
		}
	}
	app.Run(os.Args)
}

func DryRun(envars *cage.Envars) {
	log.Infof("== [DRY RUN] ==")
	d, _ := json.MarshalIndent(envars, "", "\t")
	log.Infof("envars = \n%s", string(d))
	if envars.NextTaskDefinitionArn == nil {
		log.Info("create NEXT task definition with provided json")
	}
	log.Infof("create NEXT service '%s' with desiredCount=1", *envars.NextServiceName)
	ses, err := session.NewSession(&aws.Config{
		Region: envars.Region,
	})
	if err != nil {
		log.Fatalf(err.Error())
	}
	e := ecs.New(ses)
	var (
		service *ecs.Service
	)
	if o, err := e.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: envars.Cluster,
		Services: []*string{
			envars.CurrentServiceName,
		},
	}); err != nil {
		log.Fatalf(err.Error())
	} else {
		service = o.Services[0]
	}
	log.Infof("currently %d tasks is running on service '%s'", *service.RunningCount, *envars.CurrentServiceName)
	estimated := cage.EstimateRollOutCount(*service.RunningCount)
	log.Infof("%d roll outs are expected", estimated)
}

func Action(envars *cage.Envars) error {
	ses, err := session.NewSession(&aws.Config{
		Region: envars.Region,
	})
	if err != nil {
		log.Errorf("failed to create new AWS session due to: %s", err)
		return err
	}
	ctx := &cage.Context{
		Ecs: ecs.New(ses),
		Cw:  cloudwatch.New(ses),
		Alb: elbv2.New(ses),
	}
	result, err := envars.StartGradualRollOut(ctx)
	if err != nil {
		log.Errorf("😭failed roll out new tasks due to: %s", err)
		return err
	}
	if *result.Rolledback {
		log.Warnf("🤕roll out hasn't completed successfully and rolled back to current version of service due to: %s", result.HandledError)
	} else {
		log.Infof("🎉service roll out has completed successfully!🎉")
	}
	return nil
}
