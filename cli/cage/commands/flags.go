package commands

import (
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/loilo-inc/canarycage"
	"github.com/urfave/cli"
)

func RegionFlag(dest *string) cli.Flag {
	return cli.StringFlag{
		Name:        "region",
		EnvVar:      cage.RegionKey,
		Usage:       "aws region for ecs. if not specified, try to load from aws sessions automatically",
		Destination: dest,
	}
}
func ClusterFlag(dest *string) cli.Flag {
	return cli.StringFlag{
		Name:        "cluster",
		EnvVar:      cage.ClusterKey,
		Usage:       "ecs cluster name. if not specified, load from service.json",
		Destination: dest,
	}
}
func ServiceFlag(dest *string) cli.Flag {
	return cli.StringFlag{
		Name:        "service",
		EnvVar:      cage.ServiceKey,
		Usage:       "service name. if not specified, load from service.json",
		Destination: dest,
	}
}
func TaskDefinitionArnFlag(dest *string) cli.Flag {
	return cli.StringFlag{
		Name:        "taskDefinitionArn",
		EnvVar:      cage.TaskDefinitionArnKey,
		Usage:       "full arn for next task definition. if not specified, use task-definition.json for registration",
		Destination: dest,
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
		td, svc, err := cage.LoadDefinitionsFromFiles(dir);
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
