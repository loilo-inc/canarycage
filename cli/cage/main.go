package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/loilo-inc/canarycage/v5/cli/cage/audit"
	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/cli/cage/commands"
	"github.com/loilo-inc/canarycage/v5/cli/cage/upgrade"
	"github.com/urfave/cli/v3"
)

// set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	appConf := &cageapp.App{}
	configCmdInput := func(input *cageapp.CageCmdInput) {
		input.App = appConf
	}
	upgradeCmdInput := func(input *cageapp.UpgradeCmdInput) {
		input.App = appConf
		input.CurrentVersion = version
	}
	cmds := commands.NewCageCommands(commands.ProvideCageCli)
	app := &cli.Command{
		Name:        "canarycage",
		Version:     fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
		Usage:       "A deployment tool for AWS ECS",
		Description: "A deployment tool for AWS ECS",
		Commands: []*cli.Command{
			cmds.Up(cageapp.NewCageCmdInput(os.Stdin, configCmdInput)),
			cmds.RollOut(cageapp.NewCageCmdInput(os.Stdin, configCmdInput)),
			cmds.Run(cageapp.NewCageCmdInput(os.Stdin, configCmdInput)),
			commands.Upgrade(cageapp.NewUpgradeCmdInput(upgradeCmdInput), upgrade.ProvideUpgradeDI),
			commands.Audit(appConf, audit.ProvideAuditCmd),
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "ci",
				Usage:       "CI mode. Skip all confirmations and use default values.",
				Sources:     cli.EnvVars("CI"),
				Destination: &appConf.CI,
			},
			&cli.BoolFlag{
				Name:        "no-color",
				Usage:       "Disable colored output",
				Sources:     cli.EnvVars("NO_COLOR"),
				Destination: &appConf.NoColor,
			},
		},
	}
	if err := app.Run(context.TODO(), os.Args); err != nil {
		log.Fatal(err)
	}
}
