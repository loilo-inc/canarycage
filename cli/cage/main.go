package main

import (
	"fmt"
	"log"
	"os"

	"github.com/loilo-inc/canarycage/cli/cage/audit"
	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/logos/di"
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
	configCmdInput := func(input *cageapp.CageCmdInput) {
		input.App = appConf
	}
	app := cli.NewApp()
	app.Name = "canarycage"
	app.HelpName = "cage"
	app.Version = fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)
	app.Usage = "A deployment tool for AWS ECS"
	app.Description = "A deployment tool for AWS ECS"
	cmds := commands.NewCageCommands(commands.ProvideCageCli)
	logDI := di.NewDomain(func(b *di.B) {
		b.Set(key.Logger, logger.DefaultLogger(os.Stdout, os.Stderr))
	})
	app.Commands = []*cli.Command{
		cmds.Up(cageapp.NewCageCmdInput(os.Stdin, configCmdInput)),
		cmds.RollOut(cageapp.NewCageCmdInput(os.Stdin, configCmdInput)),
		cmds.Run(cageapp.NewCageCmdInput(os.Stdin, configCmdInput)),
		commands.Upgrade(upgrade.NewUpgrader(logDI, version)),
		commands.Audit(appConf, audit.ProvideAuditCmd),
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
