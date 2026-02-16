package upgrade

import (
	"context"
	"os"

	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/logger"
	"github.com/loilo-inc/canarycage/v5/timeout"
	"github.com/loilo-inc/canarycage/v5/types"
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
