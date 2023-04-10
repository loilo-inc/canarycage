package main

import (
	"context"
	"os"

	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "canarycage"
	app.Version = "3.5.0"
	app.Description = "A gradual roll-out deployment tool for AWS ECS"
	ctx := context.Background()
	cmds := commands.NewCageCommands(ctx)
	app.Commands = cli.Commands{
		cmds.RollOut(),
		cmds.Up(),
		cmds.Run(),
	}
	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
