package cageapp

import (
	"github.com/urfave/cli/v3"
)

func RegionFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "region",
		Usage:       "aws region to be used. region specified in aws session is always ignored",
		Destination: dest,
		Required:    true,
	}
}
func ClusterFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "cluster",
		Usage:       "ecs cluster name. if not specified, load from service.json",
		Destination: dest,
	}
}
func ServiceFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "service",
		Usage:       "service name. if not specified, load from service.json",
		Destination: dest,
	}
}
func TaskDefinitionArnFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "taskDefinitionArn",
		Usage:       "full arn or family:revision of task definition. if not specified, new task definition will be created based on task-definition.json",
		Destination: dest,
	}
}

func CanaryTaskIdleDurationFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "canaryTaskIdleDuration",
		Usage:       "duration seconds for waiting canary task that isn't attached to target group considered as ready for serving traffic",
		Destination: dest,
		Value:       15,
	}
}

func TaskRunningWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "taskRunningTimeout",
		Usage:       "max duration seconds for waiting canary task running",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900, // 15 minutes
	}
}

func TaskHealthCheckWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "taskHealthCheckTimeout",
		Usage:       "max duration seconds for waiting canary task health check",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900,
	}
}

func TaskStoppedWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "taskStoppedTimeout",
		Usage:       "max duration seconds for waiting canary task stopped",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900,
	}
}

func ServiceStableWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "serviceStableTimeout",
		Usage:       "max duration seconds for waiting service stable",
		Destination: dest,
		Category:    "ADVANCED",
		Value:       900,
	}
}
