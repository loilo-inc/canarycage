package commands

import (
	"context"
	"errors"

	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/env"
	"github.com/urfave/cli/v3"
)

func Audit(app *cageapp.App, provider cageapp.AuditCmdProvider) *cli.Command {
	input := cageapp.NewAuditCmdInput()
	input.App = app
	return &cli.Command{
		Name:      "audit",
		Usage:     "Audit container images used in an ECS service",
		ArgsUsage: "[directory path of service.json]",
		Flags: []cli.Flag{
			cageapp.RegionFlag(&input.Region),
			cageapp.ClusterFlag(&input.Cluster),
			cageapp.ServiceFlag(&input.Service),
			&cli.BoolFlag{
				Name:        "detail",
				Usage:       "By default, only the name and URI of the finding are logged.",
				Value:       false,
				Destination: &input.LogDetail,
			},
			&cli.BoolFlag{
				Name:        "json",
				Usage:       "Output the audit result in JSON format",
				Value:       false,
				Destination: &input.JSON,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir, _, err := RequireArgs(cmd, 0, 1)
			if err != nil {
				return err
			}
			if input.Region == "" {
				return errors.New("--region flag is required")
			}
			if dir != "" {
				srv, err := env.LoadServiceDefinition(dir)
				if err != nil {
					return err
				}
				if srv.ServiceName == nil || srv.Cluster == nil {
					return errors.New("service.json must contain ServiceName and Cluster")
				}
				input.Service = *srv.ServiceName
				input.Cluster = *srv.Cluster
			} else if input.Cluster == "" || input.Service == "" {
				return errors.New("either directory argument or both --cluster and --service flags must be provided")
			}
			command, err := provider(ctx, input)
			if err != nil {
				return err
			}
			return command.Run(ctx)
		},
	}
}
