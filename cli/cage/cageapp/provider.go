package cageapp

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	cage "github.com/loilo-inc/canarycage"
	"github.com/loilo-inc/canarycage/cli/cage/scan"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/canarycage/task"
	"github.com/loilo-inc/canarycage/timeout"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
	"golang.org/x/xerrors"
)

func ProvideCageCli(envars *env.Envars) (types.Cage, error) {
	conf, err := loadAwsConfig(envars.Region)
	if err != nil {
		return nil, err
	}
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

func ProvideScanDI(region string) (*di.D, error) {
	conf, err := loadAwsConfig(region)
	if err != nil {
		return nil, err
	}
	d := di.NewDomain(func(b *di.B) {
		ecsCli := ecs.NewFromConfig(conf)
		ecrCli := ecr.NewFromConfig(conf)
		b.Set(key.Scanner, scan.NewScanner(ecsCli, ecrCli))
		b.Set(key.Logger, logger.DefaultLogger(os.Stdout))
	})
	return d, nil
}

func loadAwsConfig(region string) (aws.Config, error) {
	conf, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region))
	if err != nil {
		return aws.Config{}, xerrors.Errorf("failed to load aws config: %w", err)
	}
	return conf, nil
}
