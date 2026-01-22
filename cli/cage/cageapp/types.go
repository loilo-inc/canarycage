package cageapp

import (
	"context"
	"io"

	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/types"
)

type CageCmdInput struct {
	*env.Envars
	*App
	Stdin io.Reader
}

func NewCageCmdInput(stdin io.Reader, opts ...func(*CageCmdInput)) *CageCmdInput {
	input := &CageCmdInput{
		Envars: &env.Envars{},
		App:    &App{},
		Stdin:  stdin,
	}
	for _, opt := range opts {
		opt(input)
	}
	return input
}

type CageCmdProvider = func(ctx context.Context, input *CageCmdInput) (types.Cage, error)

type AuditCmdInput struct {
	*App
	Region    string
	Cluster   string
	Service   string
	LogDetail bool
}

type AuditCmdProvider = func(ctx context.Context, input *AuditCmdInput) (types.Audit, error)

func NewAuditCmdInput(opts ...func(*AuditCmdInput)) *AuditCmdInput {
	input := &AuditCmdInput{App: &App{}}
	for _, opt := range opts {
		opt(input)
	}
	return input
}

type UpgradeCmdInput struct {
	*App
	PreRelease  bool
	CurrVersion string
}
type UpgradeCmdProvider = func() (upgrade.Upgrader, error)

func NewUpgradeCmdInput(opts ...func(*UpgradeCmdInput)) *UpgradeCmdInput {
	input := &UpgradeCmdInput{App: &App{}}
	for _, opt := range opts {
		opt(input)
	}
	return input
}
