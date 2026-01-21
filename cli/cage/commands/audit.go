package commands

import (
	"errors"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/env"
	"github.com/urfave/cli/v2"
)

func Audit(provider cageapp.AuditCmdProvider) *cli.Command {
	input := cageapp.NewAuditCmdInput()
	return &cli.Command{
		Name:      "audit",
		Usage:     "Audit container images used in an ECS service",
		ArgsUsage: "[directory path of service.json and task-definition.json]",
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
		},
		Action: func(ctx *cli.Context) error {
			dir, _, err := RequireArgs(ctx, 0, 1)
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
			cmd, err := provider(ctx.Context, input)
			if err != nil {
				return err
			}
			return cmd.Run(ctx.Context)
		},
	}
}
