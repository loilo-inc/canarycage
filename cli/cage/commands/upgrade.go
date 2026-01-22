package commands

import (
	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/urfave/cli/v2"
)

func Upgrade(upgrader upgrade.Upgrader) *cli.Command {
	var preRelease bool
	return &cli.Command{
		Name:  "upgrade",
		Usage: "upgrade cage binary with the latest version",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "pre-release",
				Usage:       "include pre-release versions",
				Destination: &preRelease,
			},
		},
		Action: func(ctx *cli.Context) error {
			return upgrader.Upgrade(&upgrade.Input{
				PreRelease: preRelease,
			})
		},
	}
}
