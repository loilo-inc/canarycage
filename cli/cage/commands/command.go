package commands

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/types"
	"github.com/urfave/cli/v2"
)

type CageCommands struct {
	cageCliProvider cageapp.CageCmdProvider
}

func NewCageCommands(
	cageCliProvider cageapp.CageCmdProvider,
) *CageCommands {
	cmds := &CageCommands{cageCliProvider: cageCliProvider}
	return cmds
}

func RequireArgs(
	ctx *cli.Context,
	minArgs int,
	maxArgs int,
) (dir string, rest []string, err error) {
	if ctx.NArg() < minArgs {
		return "", nil, fmt.Errorf("invalid number of arguments. expected at least %d", minArgs)
	} else if ctx.NArg() > maxArgs {
		return "", nil, fmt.Errorf("invalid number of arguments. expected at most %d", maxArgs)
	}
	dir = ctx.Args().First()
	rest = ctx.Args().Tail()
	return
}

func (c *CageCommands) setupCage(
	input *cageapp.CageCmdInput,
	dir string,
) (types.Cage, error) {
	var service *ecs.CreateServiceInput
	var taskDefinition *ecs.RegisterTaskDefinitionInput
	if srv, err := env.LoadServiceDefinition(dir); err != nil {
		return nil, err
	} else {
		service = srv
	}
	if input.TaskDefinitionArn == "" {
		if td, err := env.LoadTaskDefinition(dir); err != nil {
			return nil, err
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
		return nil, err
	}
	cagecli, err := c.cageCliProvider(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	return cagecli, nil
}
