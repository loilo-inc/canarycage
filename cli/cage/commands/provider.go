package commands

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	cage "github.com/loilo-inc/canarycage/v5"
	"github.com/loilo-inc/canarycage/v5/awsiface"
	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/logger"
	"github.com/loilo-inc/canarycage/v5/task"
	"github.com/loilo-inc/canarycage/v5/timeout"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/loilo-inc/logos/di"
)

func ProvideCageCli(ctx context.Context, input *cageapp.CageCmdInput) (types.Cage, error) {
	conf := awsiface.MustLoadConfig(
		ctx,
		config.WithRegion(input.Region),
	)
	d := di.NewDomain(func(b *di.B) {
		b.Set(key.Env, input.Envars)
		b.Set(key.EcsCli, ecs.NewFromConfig(conf))
		b.Set(key.EcrCli, ecr.NewFromConfig(conf))
		b.Set(key.Ec2Cli, ec2.NewFromConfig(conf))
		b.Set(key.AlbCli, elasticloadbalancingv2.NewFromConfig(conf))
		b.Set(key.TaskFactory, task.NewFactory(b.Future()))
		p := logger.NewPrinter(os.Stdout, os.Stderr)
		b.Set(key.Printer, p)
		b.Set(key.Logger, logger.DefaultLogger(p))
		b.Set(key.Time, &timeout.Time{})
	})
	cagecli := cage.NewCage(d)
	return cagecli, nil
}
