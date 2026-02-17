package audit

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/loilo-inc/canarycage/v5/awsiface"
	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/logger"
	"github.com/loilo-inc/canarycage/v5/timeout"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/loilo-inc/logos/v2/di"
)

func ProvideAuditCmd(ctx context.Context, input *cageapp.AuditCmdInput) (types.Audit, error) {
	conf := awsiface.MustLoadConfig(
		ctx,
		config.WithRegion(input.Region),
	)
	d := di.NewDomain(func(b *di.B) {
		ecsCli := ecs.NewFromConfig(conf)
		ecrCli := ecr.NewFromConfig(conf)
		p := logger.NewPrinter(os.Stdout, os.Stderr)
		l := logger.DefaultLogger(p)
		b.Set(key.Scanner, NewScanner(ecsCli, ecrCli))
		b.Set(key.Printer, p)
		b.Set(key.Logger, l)
		b.Set(key.Time, &timeout.Time{})
	})
	return NewCommand(d, input), nil
}
