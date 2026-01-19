package commands

import (
	"io"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/loilo-inc/canarycage/cli/cage/prompt"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/types"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

type CageCommands struct {
	Prompt         *prompt.Prompter
	cageCliProvier cageCliProvier
}

func NewCageCommands(
	stdin io.Reader,
	cageCliProvier cageCliProvier,
) *CageCommands {
	return &CageCommands{
		Prompt:         prompt.NewPrompter(stdin),
		cageCliProvier: cageCliProvier,
	}
}

type cageCliProvier = func(e *env.Envars) (types.Cage, error)

func (c *CageCommands) requireArgs(
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

func (c *CageCommands) setupCage(
	envars *env.Envars,
	dir string,
) (types.Cage, error) {
	var service *ecs.CreateServiceInput
	var taskDefinition *ecs.RegisterTaskDefinitionInput
	if srv, err := env.LoadServiceDefinition(dir); err != nil {
		return nil, err
	} else {
		service = srv
	}
	if envars.TaskDefinitionArn == "" {
		if td, err := env.LoadTaskDefinition(dir); err != nil {
			return nil, err
		} else {
			taskDefinition = td
		}
	}
	env.MergeEnvars(envars, &env.Envars{
		Cluster:                *service.Cluster,
		Service:                *service.ServiceName,
		TaskDefinitionInput:    taskDefinition,
		ServiceDefinitionInput: service,
	})
	if err := env.EnsureEnvars(envars); err != nil {
		return nil, err
	}
	di, err := c.diProvider(envars)
	if err != nil {
		return nil, err
	}
	cagecli, err := c.cageCliProvier(di)
	if err != nil {
		return nil, err
	}
	return cagecli, nil
}
