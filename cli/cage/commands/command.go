package commands

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/env"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

func RequireArgs(
	ctx *cli.Context,
	minArgs int,
	maxArgs int,
) (dir string, rest []string, err error) {
	if ctx.NArg() < minArgs {
		return "", nil, xerrors.Errorf("invalid number of arguments. expected at least %d", minArgs)
	} else if ctx.NArg() > maxArgs {
		return "", nil, xerrors.Errorf("invalid number of arguments. expected at most %d", maxArgs)
	}
	dir = ctx.Args().First()
	rest = ctx.Args().Tail()
	return
}

func setupCage(
	ctx context.Context,
	input *cageapp.CageCmdInput,
	dir string,
) error {
	var service *ecs.CreateServiceInput
	var taskDefinition *ecs.RegisterTaskDefinitionInput
	if srv, err := env.LoadServiceDefinition(dir); err != nil {
		return err
	} else {
		service = srv
	}
	if input.TaskDefinitionArn == "" {
		if td, err := env.LoadTaskDefinition(dir); err != nil {
			return err
		} else {
			taskDefinition = td
		}
	}
	env.MergeEnvars(input.Envars, &env.Envars{
		Cluster:                *service.Cluster,
		Service:                *service.ServiceName,
		TaskDefinitionInput:    taskDefinition,
		ServiceDefinitionInput: service,
	})
	if err := env.EnsureEnvars(input.Envars); err != nil {
		return err
	}
	return nil
}
