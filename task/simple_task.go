package task

import (
	"context"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
	"golang.org/x/xerrors"
)

// simpleTask is a task that isn't attachet to any load balancer or service discovery
type simpleTask struct {
	*common
}

func NewSimpleTask(di *di.D, input *Input) Task {
	return &simpleTask{common: &common{Input: input, di: di}}
}

func (c *simpleTask) Wait(ctx context.Context) error {
	if err := c.waitForTask(ctx); err != nil {
		return err
	}
	return c.WaitForIdleDuration(ctx)
}

func (c *simpleTask) Stop(ctx context.Context) error {
	return c.stopTask(ctx)
}

func (c *simpleTask) WaitForIdleDuration(ctx context.Context) error {
	env := c.di.Get(key.Env).(*env.Envars)
	timer := c.di.Get(key.Time).(types.Time)
	log.Infof("wait %d seconds for canary task to be stable...", env.CanaryTaskIdleDuration)
	rest := env.GetCanaryTaskIdleWait()
	waitPeriod := 15 * time.Second
	for rest > 0 {
		if rest < waitPeriod {
			waitPeriod = rest
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.NewTimer(waitPeriod).C:
			rest -= waitPeriod
		}
		log.Infof("still waiting...; %d seconds left", rest)
	}
	ecsCli := c.di.Get(key.EcsCli).(awsiface.EcsClient)
	o, err := ecsCli.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &env.Cluster,
		Tasks:   []string{*c.TaskArn},
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
