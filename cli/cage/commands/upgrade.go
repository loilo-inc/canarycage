package commands

import (
	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/urfave/cli/v2"
)

func (c *CageCommands) Upgrade(
	currVersion string,
) *cli.Command {
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
			return upgrade.Upgrade(&upgrade.Input{
				CurrentVersion: currVersion,
				PreRelease:     preRelease,
			})
		},
	}
}
