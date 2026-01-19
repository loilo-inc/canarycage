package commands

import (
	"context"
	"errors"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/cli/cage/scan"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/logos/di"
	"github.com/urfave/cli/v2"
)

type diProvider = func(region string) (*di.D, error)

func Scan(diProvider diProvider) *cli.Command {
	var region string
	var cluster string
	var service string
	return &cli.Command{
		Name:      "scan",
		Usage:     "Scan ECR image vulnerabilities for the given ECS service",
		ArgsUsage: "<directory path of service.json and task-definition.json>",
		Flags: []cli.Flag{
			cageapp.RegionFlag(&region),
			cageapp.ClusterFlag(&cluster),
			cageapp.ServiceFlag(&service),
		},
		Action: func(ctx *cli.Context) error {
			dir, _, err := RequireArgs(ctx, 0, 1)
			if err != nil {
				return err
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
			d, err := diProvider(region)
			if err != nil {
				return err
			}
			scanner := d.Get(key.Scanner).(scan.Scanner)
			result, err := scanner.Scan(context.Background(), cluster, service)
			if err != nil {
				return err
			}
			logger := d.Get(key.Logger).(logger.Logger)
			printer := scan.NewPrinter(logger)
			printer.Print(result)
			return nil
		},
	}
}
