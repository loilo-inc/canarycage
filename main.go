package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/apex/log"
	"os"
)

func main() {
	envars, err := EnsureEnvars(os.Getenv, os.LookupEnv)
	ses, err := session.NewSession(&aws.Config{
		Region: &envars.Region,
	})
	if err != nil {
		log.Fatalf("failed to create new AWS session due to: %s", err.Error())
	}
	awsEcs := ecs.New(ses)
	cw := cloudwatch.New(ses)
	if err := envars.StartGradualRollOut(awsEcs, cw); err != nil {
		log.Fatalf("😭failed roll out new tasks due to: %s", err.Error())
	}
	log.Infof("🎉service roll out has completed successfully!🎉")
}
