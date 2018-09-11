package main

import (
	"github.com/urfave/cli"
	"os"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"log"
)

func main() {
	// cliのdestinationがnil pointerに代入してくれないので無効値を入れておく
	ses, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		log.Fatalf(err.Error())
	}
	app := cli.NewApp()
	app.Name = "canarycage"
	app.Version = "1.2.1"
	app.Description = "A gradual roll-out deployment tool for AWS ECS"
	app.Commands = cli.Commands{
		RollOutCommand(),
		UpCommand(ses),
	}
	app.Run(os.Args)
}
