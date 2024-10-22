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
	taskArn *string
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
			c.taskArn = o.Tasks[0].TaskArn
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
			c.taskArn = o.Tasks[0].TaskArn
		}
	}
	return nil
}

func (c *common) waitForTaskRunning(ctx context.Context) error {
	if c.taskArn == nil {
		return xerrors.New("task is not started")
	}
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)

	// NOTE: https://github.com/loilo-inc/canarycage/issues/93
	// wait for the task to be running
	time.Sleep(2 * time.Second)

	log.Infof("🥚 waiting for canary task '%s' is running...", *c.taskArn)
	if err := ecs.NewTasksRunningWaiter(ecsCli).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*c.taskArn},
	}, env.GetTaskRunningWait()); err != nil {
		return xerrors.Errorf("failed to wait for canary task to be running: %w", err)
	}
	log.Infof("🐣 canary task '%s' is running!", *c.taskArn)
	return nil
}

func (c *common) waitContainerHealthCheck(ctx context.Context) error {
	log.Infof("😷 ensuring canary task container(s) to become healthy...")
	containerHasHealthChecks := map[string]struct{}{}
	for _, definition := range c.TaskDefinition.ContainerDefinitions {
		if definition.HealthCheck != nil {
			containerHasHealthChecks[*definition.Name] = struct{}{}
		}
	}
	if len(containerHasHealthChecks) == 0 {
		log.Info("no container has health check, skipped.")
		return nil
	}
	env := c.di.Get(key.Env).(*env.Envars)
	timer := c.di.Get(key.Time).(types.Time)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	rest := env.GetTaskHealthCheckWait()
	healthCheckPeriod := 15 * time.Second
	for rest > 0 && len(containerHasHealthChecks) > 0 {
		if rest < healthCheckPeriod {
			healthCheckPeriod = rest
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.NewTimer(healthCheckPeriod).C:
			log.Infof("canary task '%s' waits until %d container(s) become healthy", *c.taskArn, len(containerHasHealthChecks))
			var task ecstypes.Task
			if o, err := ecsCli.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: &env.Cluster,
				Tasks:   []string{*c.taskArn},
			}); err != nil {
				return err
			} else {
				task = o.Tasks[0]
			}
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
		}
		rest -= healthCheckPeriod
	}
	if len(containerHasHealthChecks) == 0 {
		log.Info("🤩 canary task container(s) is healthy!")
		log.Infof("canary task '%s' ensured.", *c.taskArn)
		return nil
	}
	return xerrors.Errorf("😨 canary task hasn't become to be healthy")
}

func (c *common) stopTask(ctx context.Context) error {
	if c.taskArn == nil {
		return nil
	}
	env := c.di.Get(key.Env).(*env.Envars)
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	log.Infof("stopping the canary task '%s'...", *c.taskArn)
	if _, err := ecsCli.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: &env.Cluster,
		Task:    c.taskArn,
	}); err != nil {
		return xerrors.Errorf("failed to stop canary task: %w", err)
	}
	if err := ecs.NewTasksStoppedWaiter(ecsCli).Wait(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*c.taskArn},
	}, env.GetTaskStoppedWait()); err != nil {
		return xerrors.Errorf("failed to wait for canary task to be stopped: %w", err)
	}
	log.Infof("canary task '%s' has successfully been stopped", *c.taskArn)
	return nil
}
