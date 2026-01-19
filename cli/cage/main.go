package main

import (
	"fmt"
	"log"
	"os"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/urfave/cli/v2"
)

// set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	flag := &cageapp.Flag{}
	app := cli.NewApp()
	app.Name = "canarycage"
	app.HelpName = "cage"
	app.Version = fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)
	app.Usage = "A deployment tool for AWS ECS"
	app.Description = "A deployment tool for AWS ECS"
	cmds := commands.NewCageCommands(cageapp.ProvideCageCli)
	app.Commands = []*cli.Command{
		cmds.Up(flag),
		cmds.RollOut(flag),
		cmds.Run(flag),
		commands.Upgrade(upgrade.NewUpgrader(version)),
		commands.Scan(cageapp.ProvideScanDI),
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "ci",
			Usage:       "CI mode. Skip all confirmations and use default values.",
			EnvVars:     []string{"CI"},
			Destination: &flag.CI,
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
