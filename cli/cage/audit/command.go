package audit

import (
	"context"
	"time"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
)

type command struct {
	di           *di.D
	input        *cageapp.AuditCmdInput
	spinInterval time.Duration
}

func NewCommand(di *di.D, input *cageapp.AuditCmdInput) *command {
	return &command{
		di:           di,
		input:        input,
		spinInterval: 100 * time.Millisecond,
	}
}

func (a *command) Run(ctx context.Context) error {
	t := a.di.Get(key.Time).(types.Time)
	l := a.di.Get(key.Logger).(logger.Logger)
	scanner := a.di.Get(key.Scanner).(Scanner)
	spinner := logger.NewSpinner()
	errchannel := make(chan error, 1)
	go func() {
		defer close(errchannel)
		results, err := scanner.Scan(ctx, a.input.Cluster, a.input.Service)
		printer := NewPrinter(l, a.input.App.NoColor, a.input.LogDetail)
		l.Printf("\r") // clear spinner line
		if err != nil {
			errchannel <- err
		} else {
			printer.Print(results)
		}
	}()
	for {
		timer := t.NewTimer(a.spinInterval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errchannel:
			return err
		case <-timer.C:
			l.Printf(
				"\r%s Scanning ECR image vulnerabilities for ECS service %s/%s",
				spinner.Next(), a.input.Cluster, a.input.Service,
			)
		}
	}
}
