package upgrade

import (
	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/types"
)

func ProvideUpgradeCli(input *cageapp.UpgradeCmdInput) (types.Upgrade, error) {
	return NewUpgrader(input.CurrVersion), nil
}
