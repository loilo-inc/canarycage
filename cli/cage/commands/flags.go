package commands

import (
	cage "github.com/loilo-inc/canarycage"
	"github.com/urfave/cli/v2"
)

func RegionFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "region",
		EnvVars:     []string{cage.RegionKey},
		Usage:       "aws region for ecs. if not specified, try to load from aws sessions automatically",
		Destination: dest,
		Required:    true,
	}
}
func ClusterFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "cluster",
		EnvVars:     []string{cage.ClusterKey},
		Usage:       "ecs cluster name. if not specified, load from service.json",
		Destination: dest,
	}
}
func ServiceFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "service",
		EnvVars:     []string{cage.ServiceKey},
		Usage:       "service name. if not specified, load from service.json",
		Destination: dest,
	}
}
func TaskDefinitionArnFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "taskDefinitionArn",
		EnvVars:     []string{cage.TaskDefinitionArnKey},
		Usage:       "full arn for next task definition. if not specified, use task-definition.json for registration",
		Destination: dest,
	}
}

func CanaryTaskIdleDurationFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "canaryTaskIdleDuration",
		EnvVars:     []string{cage.CanaryTaskIdleDuration},
		Usage:       "duration seconds for waiting canary task that isn't attached to target group considered as ready for serving traffic",
		Destination: dest,
		Value:       10,
	}
}

func TaskRunningWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "taskRunningTimeout",
		EnvVars:     []string{cage.TaskRunningTimeout},
		Usage:       "max duration seconds for waiting canary task running",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900, // 15 minutes
	}
}

func TaskHealthCheckWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "taskHealthCheckTimeout",
		EnvVars:     []string{cage.TaskHealthCheckTimeout},
		Usage:       "max duration seconds for waiting canary task health check",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900,
	}
}

func TaskStoppedWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "taskStoppedTimeout",
		EnvVars:     []string{cage.TaskStoppedTimeout},
		Usage:       "max duration seconds for waiting canary task stopped",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900,
	}
}

func ServiceStableWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "serviceStableTimeout",
		EnvVars:     []string{cage.ServiceStableTimeout},
		Usage:       "max duration seconds for waiting service stable",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900,
	}
}
