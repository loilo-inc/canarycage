package commands

import (
	"context"
	"errors"

	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/scan"
	"github.com/urfave/cli/v2"
)

func (c *CageCommands) Scan(
	ecscli awsiface.EcsClient,
	ecrcli awsiface.EcrClient,
) *cli.Command {
	var region string
	var cluster string
	var service string
	return &cli.Command{
		Name:      "scan",
		Usage:     "Scan ECR image vulnerabilities for the given ECS service",
		Args:      true,
		ArgsUsage: "<directory path of service.json and task-definition.json>",
		Flags: []cli.Flag{
			RegionFlag(&region),
			ClusterFlag(&cluster),
			ServiceFlag(&service),
		},
		Action: func(ctx *cli.Context) error {
			dir, _, err := c.requireArgs(ctx, 0, 1)
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
			scanner := scan.NewScanner(ecscli, ecrcli)
			result, err := scanner.Scan(context.Background(), cluster, service)
			if err != nil {
				return err
			}
			logger := scan.DefaultLogger()
			printer := scan.NewPrinter(logger)
			printer.Print(result)
			return nil
		},
	}
}
