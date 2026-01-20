package audit

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/canarycage/timeout"
	"github.com/loilo-inc/logos/di"
)

func ProvideAuditDI(region string) (*di.D, error) {
	conf, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}
	d := di.NewDomain(func(b *di.B) {
		ecsCli := ecs.NewFromConfig(conf)
		ecrCli := ecr.NewFromConfig(conf)
		b.Set(key.Scanner, NewScanner(ecsCli, ecrCli))
		b.Set(key.Logger, logger.DefaultLogger(os.Stdout))
		b.Set(key.Time, &timeout.Time{})
	})
	return d, nil
}
