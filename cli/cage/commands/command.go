package commands

import (
	"context"
	"github.com/urfave/cli"
)

type CageCommands interface {
	Up() cli.Command
	RollOut() cli.Command
}

type cageCommands struct {
	ctx context.Context
}

func NewCageCommands(ctx context.Context) CageCommands {
	return &cageCommands{ctx: ctx}
}
