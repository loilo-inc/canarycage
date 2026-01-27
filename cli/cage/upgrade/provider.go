package upgrade

import (
	"context"
	"os"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/key"
	"github.com/loilo-inc/canarycage/logger"
	"github.com/loilo-inc/canarycage/timeout"
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
)

func ProvideUpgradeDI(ctx context.Context, input *cageapp.UpgradeCmdInput) (types.Upgrade, error) {
	d := di.NewDomain(
		func(b *di.B) {
			p := logger.NewPrinter(os.Stdout, os.Stderr)
			l := logger.DefaultLogger(p)
			b.Set(key.Printer, p)
			b.Set(key.Logger, l)
			b.Set(key.Time, &timeout.Time{})
		})
	return NewUpgrader(d, input), nil
}
