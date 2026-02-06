package commands

import (
	"github.com/loilo-inc/canarycage/cli/cage/policy"
	"github.com/urfave/cli/v2"
)

func Policy() *cli.Command {
	var short bool
	return &cli.Command{
		Name:  "policy",
		Usage: "output IAM policy required for canarycage",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "short",
				Usage:       "output short format",
				Destination: &short,
			},
		},
		Action: func(ctx *cli.Context) error {
			cmd := policy.NewCommand(ctx.App.Writer, short)
			return cmd.Run()
		},
	}
}
