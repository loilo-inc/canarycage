package audit

import (
	"context"

	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/logger"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/loilo-inc/logos/di"
)

type command struct {
	*cageapp.AuditCmdInput
	di *di.D
}

func NewCommand(di *di.D, input *cageapp.AuditCmdInput) *command {
	return &command{
		di:            di,
		AuditCmdInput: input,
	}
}

func (a *command) Run(ctx context.Context) (err error) {
	results, err := a.doScan(ctx)
	if err != nil {
		return err
	}
	p := NewPrinter(a.di, a.NoColor, a.LogDetail)
	if a.JSON {
		metadata := Target{
			Region:  a.Region,
			Cluster: a.Cluster,
			Service: a.Service,
		}
		p.PrintJSON(metadata, results)
	} else {
		p.Print(results)
	}
	return nil
}

func (a *command) doScan(ctx context.Context) (results []ScanResult, err error) {
	l := a.di.Get(key.Printer).(logger.Printer)
	t := a.di.Get(key.Time).(types.Time)
	defer l.PrintErrf("\r")
	waiter := make(chan struct{}, 1)
	spinner := logger.NewSpinner()
	go func() {
		defer close(waiter)
		scanner := a.di.Get(key.Scanner).(Scanner)
		results, err = scanner.Scan(ctx, a.Cluster, a.Service)
		waiter <- struct{}{}
	}()
	interval := a.SpinInterval()
	for {
		timer := t.NewTimer(interval)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-waiter:
			return
		case <-timer.C:
			l.PrintErrf("\r%s", spinner.Next())
		}
	}
}
