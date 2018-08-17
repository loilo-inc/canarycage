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
)

func main() {
	// cliのdestinationがnil pointerに代入してくれないので無効値を入れておく
	envars := &cage.Envars{
		Region:                   aws.String(""),
		Cluster:                  aws.String(""),
		LoadBalancerArn:          aws.String(""),
		NextServiceName:          aws.String(""),
		CurrentServiceName:       aws.String(""),
		NextTaskDefinitionBase64: aws.String(""),
		NextTaskDefinitionArn:    aws.String(""),
		AvailabilityThreshold:    aws.Float64(-1.0),
		ResponseTimeThreshold:    aws.Float64(-1.0),
		RollOutPeriod:            aws.Int64(-1),
	}
	app := cli.NewApp()
	app.Name = "canarycage"
	app.Version = "0.0.1"
	app.Description = "A gradual roll-out deployment tool for AWS ECS"
	app.Flags = []cli.Flag{
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
			Name:        "loadBalancerArn",
			EnvVar:      cage.LoadBalancerArnKey,
			Usage:       "full arn of service load balancer",
			Destination: envars.LoadBalancerArn,
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
			Destination: envars.NextTaskDefinitionBase64,
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
	}
	app.Action = func(ctx *cli.Context) {
		err := cage.EnsureEnvars(envars)
		if err != nil {
			log.Fatalf(err.Error())
		}
		if err := Action(envars); err != nil {
			log.Fatalf("failed: %s", err)
		}
	}
	app.Run(os.Args)
}

func Action(envars *cage.Envars) error {
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
