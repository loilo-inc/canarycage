package main

import (
	"log"
	"os"

	cage "github.com/loilo-inc/canarycage"
	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "canarycage"
	app.Version = "4.0.0-rc1"
	app.Usage = "A deployment tool for AWS ECS"
	app.Description = "A deployment tool for AWS ECS"
	envars := cage.Envars{}
	cmds := commands.NewCageCommands(os.Stdin, commands.DefalutCageCliProvider)
	app.Commands = cmds.Commands(&envars)
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "ci",
			Usage:       "CI mode. Skip all confirmations and use default values.",
			EnvVars:     []string{"CI"},
			Destination: &envars.CI,
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
