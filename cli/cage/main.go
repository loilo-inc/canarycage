package main

import (
	"context"
	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/urfave/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "canarycage"
	app.Version = "3.0.1-alpha1"
	app.Description = "A gradual roll-out deployment tool for AWS ECS"
	ctx := context.Background()
	cmds := commands.NewCageCommands(ctx)
	app.Commands = cli.Commands{
		cmds.RollOut(),
		cmds.Up(),
	}
	app.Run(os.Args)
}
