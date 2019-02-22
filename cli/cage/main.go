package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/urfave/cli"
	"log"
	"os"
)

func main() {
	ses, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	if err != nil {
		log.Fatalf(err.Error())
	}
	app := cli.NewApp()
	app.Name = "canarycage"
	app.Version = "3.0.0-alpha1"
	app.Description = "A gradual roll-out deployment tool for AWS ECS"
	ctx := context.Background()
	cmds := commands.NewCageCommands(&commands.CageCommandsInput{
		GlobalContext:ctx,
		Session: ses,
	})
	app.Commands = cli.Commands{
		cmds.RollOut(),
		cmds.Up(),
	}
	app.Run(os.Args)
}
