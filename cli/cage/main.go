package main

import (
	"fmt"
	"log"
	"os"

	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/loilo-inc/canarycage/env"
	"github.com/urfave/cli/v2"
)

// set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app := cli.NewApp()
	app.Name = "canarycage"
	app.HelpName = "cage"
	app.Version = fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)
	app.Usage = "A deployment tool for AWS ECS"
	app.Description = "A deployment tool for AWS ECS"
	envars := env.Envars{}
	cmds := commands.NewCageCommands(os.Stdin, commands.DefalutCageCliProvider)
	app.Commands = []*cli.Command{
		cmds.Up(&envars),
		cmds.RollOut(&envars),
		cmds.Run(&envars),
		cmds.Upgrade(upgrade.NewUpgrader(version)),
	}
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
