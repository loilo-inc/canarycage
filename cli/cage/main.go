package main

import (
	"context"
	"log"
	"os"

	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "canarycage"
	app.Version = "3.7.0"
	app.Description = "A gradual roll-out deployment tool for AWS ECS"
	ctx := context.Background()
	cmds := commands.NewCageCommands(ctx, os.Stdin)
	app.Commands = cmds.Commands()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
