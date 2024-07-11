package commands_test

import (
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/loilo-inc/canarycage/cli/cage/upgrade"
	"github.com/loilo-inc/canarycage/mocks/mock_upgrade"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"
)

func TestUpgrade(t *testing.T) {
	t.Run("Upgrade", func(t *testing.T) {
		app := cli.NewApp()
		ctrl := gomock.NewController(t)
		u := mock_upgrade.NewMockUpgrader(ctrl)
		u.EXPECT().Upgrade(
			gomock.Eq(&upgrade.Input{}),
		).Return(nil)
		cmds := commands.NewCageCommands(nil, nil)
		app.Commands = []*cli.Command{
			cmds.Upgrade(u),
		}
		err := app.Run([]string{"cage", "upgrade"})
		assert.NoError(t, err)
	})
	t.Run("Upgrade with pre-release", func(t *testing.T) {
		app := cli.NewApp()
		ctrl := gomock.NewController(t)
		u := mock_upgrade.NewMockUpgrader(ctrl)
		u.EXPECT().Upgrade(
			gomock.Eq(&upgrade.Input{PreRelease: true}),
		).Return(nil)
		cmds := commands.NewCageCommands(nil, nil)
		app.Commands = []*cli.Command{
			cmds.Upgrade(u),
		}
		err := app.Run([]string{"cage", "upgrade", "--pre-release"})
		assert.NoError(t, err)
	})
}
