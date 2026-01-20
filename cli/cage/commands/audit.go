package commands

import (
	"errors"

	"github.com/loilo-inc/canarycage/cli/cage/audit"
	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/logos/di"
	"github.com/urfave/cli/v2"
)

type diProvider = func(region string) (*di.D, error)

func Audit(app *cageapp.App, diProvider diProvider) *cli.Command {
	var region string
	var cluster string
	var service string
	var logDetail bool
	return &cli.Command{
		Name:      "audit",
		Usage:     "Audit container images used in an ECS service",
		ArgsUsage: "[directory path of service.json and task-definition.json]",
		Flags: []cli.Flag{
			cageapp.RegionFlag(&region),
			cageapp.ClusterFlag(&cluster),
			cageapp.ServiceFlag(&service),
			&cli.BoolFlag{
				Name:        "detail",
				Usage:       "By default, only the name and URI of the finding are logged.",
				Value:       false,
				Destination: &logDetail,
			},
		},
		Action: func(ctx *cli.Context) error {
			dir, _, err := RequireArgs(ctx, 0, 1)
			if err != nil {
				return err
			}
			if region == "" {
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
				service = *srv.ServiceName
				cluster = *srv.Cluster
			} else if cluster == "" || service == "" {
				return errors.New("either directory argument or both --cluster and --service flags must be provided")
			}
			di, err := diProvider(region)
			if err != nil {
				return err
			}
			cmd := audit.NewCommand(di, app, logDetail)
			return cmd.Run(ctx.Context, cluster, service)
		},
	}
}
