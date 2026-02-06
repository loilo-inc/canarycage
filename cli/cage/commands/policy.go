package commands

import (
	"github.com/loilo-inc/canarycage/cli/cage/policy"
	"github.com/urfave/cli/v2"
)

func Policy() *cli.Command {
	var pretty bool
	return &cli.Command{
		Name:  "policy",
		Usage: "output IAM policy required for canarycage",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "pretty",
				Usage:       "output indented JSON",
				Destination: &pretty,
			},
		},
		Action: func(ctx *cli.Context) error {
			cmd := policy.NewCommand(ctx.App.Writer, pretty)
			return cmd.Run()
		},
	}
}
