package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/urfave/cli"
	"log"
	"os"
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
	app.Version = "2.0.0-alpha5"
	app.Description = "A gradual roll-out deployment tool for AWS ECS"
	app.Commands = cli.Commands{
		commands.RollOutCommand(),
		commands.UpCommand(ses),
	}
	app.Run(os.Args)
}
