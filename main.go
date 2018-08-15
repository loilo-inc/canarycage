package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/apex/log"
	"github.com/urfave/cli"
	"os"
)

func main() {
	envars := &Envars{}
	app := cli.NewApp()
	app.Name = "canarycage"
	app.Version = "0.0.1"
	app.Description = "A gradual roll-out deployment tool for AWS ECS"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "region",
			EnvVar:      kRegionKey,
			Value:       "us-west-2",
			Usage:       "aws region for ecs",
			Destination: envars.Region,
		},
		cli.StringFlag{
			Name:        "cluster",
			EnvVar:      kClusterKey,
			Usage:       "ecs cluster name",
			Destination: envars.Cluster,
		},
		cli.StringFlag{
			Name:        "loadBalancerArn",
			EnvVar:      kLoadBalancerArnKey,
			Usage:       "full arn of service load balancer",
			Destination: envars.LoadBalancerArn,
		},
		cli.StringFlag{
			Name:        "nextServiceName",
			EnvVar:      kNextServiceNameKey,
			Usage:       "next service name",
			Destination: envars.NextServiceName,
		},
		cli.StringFlag{
			Name:        "currentServiceName",
			EnvVar:      kCurrentServiceNameKey,
			Usage:       "current service name",
			Destination: envars.CurrentServiceName,
		},
		cli.StringFlag{
			Name:        "nextServiceDefinitionBase64",
			EnvVar:      kNextServiceDefinitionBase64Key,
			Usage:       "base64 encoded service definition for next service",
			Destination: envars.NextTaskDefinitionBase64,
		},
		cli.StringFlag{
			Name:        "nextTaskDefinitionBase64",
			EnvVar:      kNextTaskDefinitionBase64Key,
			Usage:       "base64 encoded task definition for next task definition",
			Destination: envars.NextTaskDefinitionBase64,
		},
		cli.Float64Flag{
			Name:        "availabilityThreshold",
			EnvVar:      kAvailabilityThresholdKey,
			Usage:       "availability (request success rate) threshold used to evaluate service health by CloudWatch",
			Value:       0.9970,
			Destination: envars.AvailabilityThreshold,
		},
		cli.Float64Flag{
			Name:        "responseTimeThreshold",
			EnvVar:      kResponseTimeThresholdKey,
			Usage:       "average response time (sec) threshold used to evaluate service health by CloudWatch",
			Value:       1.0,
			Destination: envars.ResponseTimeThreshold,
		},
		cli.Int64Flag{
			Name:        "rollOutPeriod",
			EnvVar:      kRollOutPeriodKey,
			Usage:       "each roll out period (sec)",
			Value:       300,
			Destination: envars.RollOutPeriod,
		},
		cli.Int64Flag{
			Name:        "updateServicePeriod",
			EnvVar:      kUpdateServicePeriod,
			Usage:       "period (sec) of waiting for update-service result",
			Value:       60,
			Destination: envars.RollOutPeriod,
		},
		cli.Int64Flag{
			Name:        "updateServiceTimeout",
			EnvVar:      kUpdateServiceTimeout,
			Usage:       "timeout (sec) of waiting for update-service result",
			Value:       300,
			Destination: envars.RollOutPeriod,
		},

	}
	app.Action = func(ctx *cli.Context) {
		err := EnsureEnvars(envars)
		if err != nil {
			log.Fatalf(err.Error())
		}
		if err := Action(envars); err != nil {
			log.Fatalf("failed: %s", err)
		}
	}
	app.Run(os.Args)
}

func Action(envars *Envars) error {
	ses, err := session.NewSession(&aws.Config{
		Region: envars.Region,
	})
	if err != nil {
		log.Errorf("failed to create new AWS session due to: %s", err)
		return err
	}
	awsEcs := ecs.New(ses)
	cw := cloudwatch.New(ses)
	if err := envars.StartGradualRollOut(awsEcs, cw); err != nil {
		log.Errorf("😭failed roll out new tasks due to: %s", err)
		return err
	}
	log.Infof("🎉service roll out has completed successfully!🎉")
	return nil
}
