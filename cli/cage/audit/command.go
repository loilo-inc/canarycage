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

func (a *command) Run(ctx context.Context) (err error) {
	results, err := a.doScan(ctx)
	if err != nil {
		return err
	}
	p := NewPrinter(a.di, a.input.NoColor, a.input.LogDetail)
	if a.input.JSON {
		metadata := &Target{
			Region:  a.input.Region,
			Cluster: a.input.Cluster,
			Service: a.input.Service,
		}
		p.PrintJSON(metadata, results)
	} else {
		p.Print(results)
	}
	return nil
}

func (a *command) doScan(ctx context.Context) (results []*ScanResult, err error) {
	l := a.di.Get(key.Logger).(logger.Logger)
	t := a.di.Get(key.Time).(types.Time)
	defer l.Errorf("\r")
	waiter := make(chan struct{}, 1)
	spinner := logger.NewSpinner()
	go func() {
		defer close(waiter)
		scanner := a.di.Get(key.Scanner).(Scanner)
		results, err = scanner.Scan(ctx, a.input.Cluster, a.input.Service)
		waiter <- struct{}{}
	}()
	for {
		timer := t.NewTimer(a.spinInterval)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-waiter:
			return
		case <-timer.C:
			l.Errorf("\r%s", spinner.Next())
		}
	}
}
