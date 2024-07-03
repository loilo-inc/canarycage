package task

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
	"golang.org/x/xerrors"
)

type Input struct {
	TaskDefinition       *ecstypes.TaskDefinition
	NetworkConfiguration *ecstypes.NetworkConfiguration
	PlatformVersion      *string
}

type common struct {
	*Input
	di      *di.D
	TaskArn *string
}

func (c *common) Start(ctx context.Context) error {
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	group := fmt.Sprintf("cage:canary-task:%s", env.Service)
	if env.CanaryInstanceArn != "" {
		// ec2
		if o, err := ecsCli.StartTask(ctx, &ecs.StartTaskInput{
			Cluster:              &env.Cluster,
			Group:                &group,
			NetworkConfiguration: c.NetworkConfiguration,
			TaskDefinition:       c.TaskDefinition.TaskDefinitionArn,
			ContainerInstances:   []string{env.CanaryInstanceArn},
		}); err != nil {
			return err
		} else {
			c.TaskArn = o.Tasks[0].TaskArn
		}
	} else {
		// fargate
		if o, err := ecsCli.RunTask(ctx, &ecs.RunTaskInput{
			Cluster:              &env.Cluster,
			Group:                &group,
			NetworkConfiguration: c.NetworkConfiguration,
			TaskDefinition:       c.TaskDefinition.TaskDefinitionArn,
			LaunchType:           ecstypes.LaunchTypeFargate,
			PlatformVersion:      c.PlatformVersion,
		}); err != nil {
			return err
		} else {
			c.TaskArn = o.Tasks[0].TaskArn
		}
	}
	return nil
}

func (c *common) waitForTask(ctx context.Context) error {
	if c.TaskArn == nil {
		return xerrors.New("task is not started")
	}
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	log.Infof("🥚 waiting for canary task '%s' is running...", *c.TaskArn)
	if err := ecs.NewTasksRunningWaiter(ecsCli).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*c.TaskArn},
	}, env.GetTaskRunningWait()); err != nil {
		return err
	}
	log.Infof("🐣 canary task '%s' is running!", *c.TaskArn)
	if err := c.waitContainerHealthCheck(ctx); err != nil {
		return err
	}
	log.Info("🤩 canary task container(s) is healthy!")
	log.Infof("canary task '%s' ensured.", *c.TaskArn)
	return nil
}

func (c *common) waitContainerHealthCheck(ctx context.Context) error {
	log.Infof("😷 ensuring canary task container(s) to become healthy...")
	env := c.di.Get(key.Env).(*env.Envars)
	timer := c.di.Get(key.Time).(types.Time)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	containerHasHealthChecks := map[string]struct{}{}
	for _, definition := range c.TaskDefinition.ContainerDefinitions {
		if definition.HealthCheck != nil {
			containerHasHealthChecks[*definition.Name] = struct{}{}
		}
	}
	rest := env.GetTaskHealthCheckWait()
	healthCheckPeriod := 15 * time.Second
	for rest > 0 {
		if rest < healthCheckPeriod {
			healthCheckPeriod = rest
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.NewTimer(healthCheckPeriod).C:
			log.Infof("canary task '%s' waits until %d container(s) become healthy", *c.TaskArn, len(containerHasHealthChecks))
			if o, err := ecsCli.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: &env.Cluster,
				Tasks:   []string{*c.TaskArn},
			}); err != nil {
				return err
			} else {
				task := o.Tasks[0]
				if *task.LastStatus != "RUNNING" {
					return xerrors.Errorf("😫 canary task has stopped: %s", *task.StoppedReason)
				}
				for _, container := range task.Containers {
					if _, ok := containerHasHealthChecks[*container.Name]; !ok {
						continue
					}
					if container.HealthStatus != ecstypes.HealthStatusHealthy {
						log.Infof("container '%s' is not healthy: %s", *container.Name, container.HealthStatus)
						continue
					}
					delete(containerHasHealthChecks, *container.Name)
				}
				if len(containerHasHealthChecks) == 0 {
					return nil
				}
			}
		}
		rest -= healthCheckPeriod
	}
	return xerrors.Errorf("😨 canary task hasn't become to be healthy")
}

func (c *common) stopTask(ctx context.Context) error {
	if c.TaskArn == nil {
		return nil
	}
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	log.Infof("stopping the canary task '%s'...", *c.TaskArn)
	if _, err := ecsCli.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &env.Cluster,
		Task:    c.TaskArn,
	}); err != nil {
		return xerrors.Errorf("failed to stop canary task: %w", err)
	}
	if err := ecs.NewTasksStoppedWaiter(ecsCli).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*c.TaskArn},
	}, env.GetTaskStoppedWait()); err != nil {
		return xerrors.Errorf("failed to wait for canary task to be stopped: %w", err)
	}
	log.Infof("canary task '%s' has successfully been stopped", *c.TaskArn)
	return nil
}
