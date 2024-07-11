package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	cage "github.com/loilo-inc/canarycage"
	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/timeout"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

// set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app := cli.NewApp()
	app.Name = "canarycage"
	app.HelpName = "cage"
	app.Version = fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)
	app.Usage = "A deployment tool for AWS ECS"
	app.Description = "A deployment tool for AWS ECS"
	envars := env.Envars{}
	cmds := commands.NewCageCommands(os.Stdin, provideCageCli)
	app.Commands = []*cli.Command{
		cmds.Up(&envars),
		cmds.RollOut(&envars),
		cmds.Run(&envars),
		cmds.Upgrade(upgrade.NewUpgrader(version)),
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "ci",
			Usage:       "CI mode. Skip all confirmations and use default values.",
			EnvVars:     []string{"CI"},
			Destination: &envars.CI,
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func provideCageCli(envars *env.Envars) (types.Cage, error) {
	conf, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(envars.Region))
	if err != nil {
		return nil, xerrors.Errorf("failed to load aws config: %w", err)
	}
	d := di.NewDomain(func(b *di.B) {
		b.Set(key.Env, envars)
		b.Set(key.EcsCli, ecs.NewFromConfig(conf))
		b.Set(key.Ec2Cli, ec2.NewFromConfig(conf))
		b.Set(key.AlbCli, elasticloadbalancingv2.NewFromConfig(conf))
		b.Set(key.TaskFactory, task.NewFactory(b.Future()))
		b.Set(key.Time, &timeout.Time{})
	})
	cagecli := cage.NewCage(d)
	return cagecli, nil
}
