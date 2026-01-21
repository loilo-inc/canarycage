package commands

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	cage "github.com/loilo-inc/canarycage"
	"github.com/loilo-inc/canarycage/awsiface"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/timeout"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
)

func ProvideCageCli(ctx context.Context, envars *env.Envars) (types.Cage, error) {
	conf := awsiface.MustLoadConfig(
		ctx,
		config.WithRegion(envars.Region),
	)
	d := di.NewDomain(func(b *di.B) {
		b.Set(key.Env, envars)
		b.Set(key.EcsCli, ecs.NewFromConfig(conf))
		b.Set(key.EcrCli, ecr.NewFromConfig(conf))
		b.Set(key.Ec2Cli, ec2.NewFromConfig(conf))
		b.Set(key.AlbCli, elasticloadbalancingv2.NewFromConfig(conf))
		b.Set(key.TaskFactory, task.NewFactory(b.Future()))
		b.Set(key.Time, &timeout.Time{})
	})
	cagecli := cage.NewCage(d)
	return cagecli, nil
}
