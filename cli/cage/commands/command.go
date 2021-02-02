package commands

import (
	"context"
	"github.com/urfave/cli/v2"
)

type CageCommands interface {
	Up() *cli.Command
	RollOut() *cli.Command
	Run() *cli.Command
}

type cageCommands struct {
	ctx context.Context
}

func NewCageCommands(ctx context.Context) CageCommands {
	return &cageCommands{ctx: ctx}
}
