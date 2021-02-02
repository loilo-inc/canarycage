package commands

import (
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/loilo-inc/canarycage"
	"github.com/urfave/cli/v2"
)

func RegionFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "region",
		EnvVars:      []string{cage.RegionKey},
		Usage:       "aws region for ecs. if not specified, try to load from aws sessions automatically",
		Destination: dest,
	}
}
func ClusterFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "cluster",
		EnvVars:      []string{cage.ClusterKey},
		Usage:       "ecs cluster name. if not specified, load from service.json",
		Destination: dest,
	}
}
func ServiceFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "service",
		EnvVars:      []string{cage.ServiceKey},
		Usage:       "service name. if not specified, load from service.json",
		Destination: dest,
	}
}
func TaskDefinitionArnFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "taskDefinitionArn",
		EnvVars:      []string{cage.TaskDefinitionArnKey},
		Usage:       "full arn for next task definition. if not specified, use task-definition.json for registration",
		Destination: dest,
	}
}

func CanaryTaskIdleDurationFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "canaryTaskIdleDuration",
		EnvVars:      []string{cage.CanaryTaskIdleDuration},
		Usage:       "Idle duration seconds for ensuring canary task that has no attached load balancer",
		Destination: dest,
		Value:       10,
	}
}

func (c *cageCommands) aggregateEnvars(
	ctx *cli.Context,
	envars *cage.Envars,
) {
	var _region string
	ses, err := session.NewSession()
	if err != nil {
		log.Fatalf(err.Error())
	}
	if envars.Region != "" {
		_region = envars.Region
		log.Infof("🗺 region was set: %s", _region)
	} else if *ses.Config.Region != "" {
		_region = *ses.Config.Region
		log.Infof("🗺 region was loaded from sessions: %s", _region)
	} else {
		log.Fatalf("🙄 region must specified by --region flag or aws session")
	}
	if ctx.NArg() > 0 {
		dir := ctx.Args().Get(0)
		td, svc, err := cage.LoadDefinitionsFromFiles(dir)
		if err != nil {
			log.Fatalf(err.Error())
		}
		cage.MergeEnvars(envars, &cage.Envars{
			Cluster:                *svc.Cluster,
			Service:                *svc.ServiceName,
			TaskDefinitionInput:    td,
			ServiceDefinitionInput: svc,
		})
	}
	if err := cage.EnsureEnvars(envars); err != nil {
		log.Fatalf(err.Error())
	}
}
