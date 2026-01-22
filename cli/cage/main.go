package main

import (
	"fmt"
	"log"
	"os"

	"github.com/loilo-inc/canarycage/cli/cage/audit"
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
	appConf := &cageapp.App{}
	cageCmdInput := cageapp.NewCageCmdInput(os.Stdin, func(input *cageapp.CageCmdInput) {
		input.App = appConf
	})
	auditCmdInput := cageapp.NewAuditCmdInput(func(input *cageapp.AuditCmdInput) {
		input.App = appConf
	})
	upgradeCmdInput := cageapp.NewUpgradeCmdInput(func(input *cageapp.UpgradeCmdInput) {
		input.App = appConf
	})
	app := cli.NewApp()
	app.Name = "canarycage"
	app.HelpName = "cage"
	app.Version = fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)
	app.Usage = "A deployment tool for AWS ECS"
	app.Description = "A deployment tool for AWS ECS"
	app.Commands = []*cli.Command{
		commands.Up(cageCmdInput, commands.ProvideCageCli),
		commands.RollOut(cageCmdInput, commands.ProvideCageCli),
		commands.Run(cageCmdInput, commands.ProvideCageCli),
		commands.Upgrade(upgradeCmdInput, upgrade.ProvideUpgrader),
		commands.Audit(auditCmdInput, audit.ProvideAuditCli),
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "ci",
			Usage:       "CI mode. Skip all confirmations and use default values.",
			EnvVars:     []string{"CI"},
			Destination: &appConf.CI,
		},
		&cli.BoolFlag{
			Name:        "no-color",
			Usage:       "Disable colored output",
			EnvVars:     []string{"NO_COLOR"},
			Destination: &appConf.NoColor,
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
