package commands

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/env"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/urfave/cli/v3"
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
	cmd *cli.Command,
	minArgs int,
	maxArgs int,
) (dir string, rest []string, err error) {
	if cmd.NArg() < minArgs {
		return "", nil, fmt.Errorf("invalid number of arguments. expected at least %d", minArgs)
	} else if cmd.NArg() > maxArgs {
		return "", nil, fmt.Errorf("invalid number of arguments. expected at most %d", maxArgs)
	}
	dir = cmd.Args().First()
	rest = cmd.Args().Tail()
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
