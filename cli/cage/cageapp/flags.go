package cageapp

import (
	"github.com/loilo-inc/canarycage/env"
	"github.com/urfave/cli/v2"
)

func RegionFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "region",
		EnvVars:     []string{env.RegionKey},
		Usage:       "aws region for ecs. if not specified, try to load from aws sessions automatically",
		Destination: dest,
		Required:    true,
	}
}
func ClusterFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "cluster",
		EnvVars:     []string{env.ClusterKey},
		Usage:       "ecs cluster name. if not specified, load from service.json",
		Destination: dest,
	}
}
func ServiceFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "service",
		EnvVars:     []string{env.ServiceKey},
		Usage:       "service name. if not specified, load from service.json",
		Destination: dest,
	}
}
func TaskDefinitionArnFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "taskDefinitionArn",
		EnvVars:     []string{env.TaskDefinitionArnKey},
		Usage:       "full arn or family:revision of task definition. if not specified, new task definition will be created based on task-definition.json",
		Destination: dest,
	}
}

func CanaryTaskIdleDurationFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "canaryTaskIdleDuration",
		EnvVars:     []string{env.CanaryTaskIdleDuration},
		Usage:       "duration seconds for waiting canary task that isn't attached to target group considered as ready for serving traffic",
		Destination: dest,
		Value:       15,
	}
}

func TaskRunningWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "taskRunningTimeout",
		EnvVars:     []string{env.TaskRunningTimeout},
		Usage:       "max duration seconds for waiting canary task running",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900, // 15 minutes
	}
}

func TaskHealthCheckWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "taskHealthCheckTimeout",
		EnvVars:     []string{env.TaskHealthCheckTimeout},
		Usage:       "max duration seconds for waiting canary task health check",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900,
	}
}

func TaskStoppedWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "taskStoppedTimeout",
		EnvVars:     []string{env.TaskStoppedTimeout},
		Usage:       "max duration seconds for waiting canary task stopped",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900,
	}
}

func ServiceStableWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "serviceStableTimeout",
		EnvVars:     []string{env.ServiceStableTimeout},
		Usage:       "max duration seconds for waiting service stable",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900,
	}
}
