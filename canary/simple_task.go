package canary

import (
	"context"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"golang.org/x/xerrors"
)

type simpleTask struct {
	*common
}

func (c *simpleTask) Wait(ctx context.Context) error {
	if err := c.wait(ctx); err != nil {
		return err
	}
	return c.waitForIdleDuration(ctx)
}

func (c *simpleTask) Stop(ctx context.Context) error {
	return c.stopTask(ctx)
}

func (c *simpleTask) waitForIdleDuration(ctx context.Context) error {
	log.Infof("wait %d seconds for canary task to be stable...", c.Env.CanaryTaskIdleDuration)
	duration := c.Env.CanaryTaskIdleDuration
	for duration > 0 {
		wt := 10
		if duration < 10 {
			wt = duration
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.Time.NewTimer(time.Duration(wt) * time.Second).C:
			duration -= 10
		}
		log.Infof("still waiting...; %d seconds left", duration)
	}
	o, err := c.Ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &c.Env.Cluster,
		Tasks:   []string{*c.taskArn},
	})
	if err != nil {
		return err
	}
	task := o.Tasks[0]
	if *task.LastStatus != "RUNNING" {
		return xerrors.Errorf("😫 canary task has stopped: %s", *task.StoppedReason)
	}
	return nil
}
