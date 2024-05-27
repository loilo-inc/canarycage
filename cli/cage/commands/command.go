package commands

import (
	"context"
	"io"

	"github.com/loilo-inc/canarycage/cli/cage/prompt"
	"github.com/urfave/cli/v2"
)

type CageCommands interface {
	Commands() []*cli.Command
}

type cageCommands struct {
	ctx    context.Context
	prompt *prompt.Prompter
}

func NewCageCommands(ctx context.Context, stdin io.Reader) CageCommands {
	return &cageCommands{ctx: ctx,
		prompt: prompt.NewPrompter(stdin),
	}
}

func (c *cageCommands) Commands() []*cli.Command {
	return []*cli.Command{
		c.Up(),
		c.RollOut(),
		c.Run(),
		c.Recreate(),
	}
}
