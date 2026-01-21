package cageapp

import (
	"context"

	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/types"
)

type CageCmdProvider = func(ctx context.Context, e *env.Envars) (types.Cage, error)

type AuditCmdInput struct {
	App       *App
	Region    string
	Cluster   string
	Service   string
	LogDetail bool
}
type AuditCmdProvider = func(ctx context.Context, input *AuditCmdInput) (types.Audit, error)
