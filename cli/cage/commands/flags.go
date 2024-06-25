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
		Usage:       "Idle duration seconds for ensuring canary task that has no attached load balancer",
		Destination: dest,
		Value:       10,
	}
}

func TaskRunningWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "canaryTaskRunningWait",
		EnvVars:     []string{cage.CanaryTaskRunningWait},
		Usage:       "Duration seconds for waiting canary task running",
		Destination: dest,
		Value:       300,
	}
}

func TaskHealthCheckWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "canaryTaskHealthCheckWait",
		EnvVars:     []string{cage.CanaryTaskHealthCheckWait},
		Usage:       "Duration seconds for waiting canary task health check",
		Destination: dest,
		Value:       300,
	}
}

func TaskStoppedWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "canaryTaskStoppedWait",
		EnvVars:     []string{cage.CanaryTaskStoppedWait},
		Usage:       "Duration seconds for waiting canary task stopped",
		Destination: dest,
		Value:       300,
	}
}

func ServiceStableWaitFlag(dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        "serviceStableWait",
		EnvVars:     []string{cage.ServiceStableWait},
		Usage:       "Duration seconds for waiting service stable",
		Destination: dest,
		Value:       300,
	}
}
