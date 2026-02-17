package commands

import (
	"context"

	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/urfave/cli/v3"
)

func Upgrade(input *cageapp.UpgradeCmdInput, provider cageapp.UpgradeCmdProvider,
) *cli.Command {
	return &cli.Command{
		Name:  "upgrade",
		Usage: "upgrade cage binary with the latest version",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "pre-release",
				Usage:       "include pre-release versions",
				Destination: &input.PreRelease,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			upgrader, err := provider(ctx, input)
			if err != nil {
				return err
			}
			return upgrader.Upgrade(ctx)
		},
	}
}
