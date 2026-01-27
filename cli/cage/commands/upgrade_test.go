package commands_test

import (
	"context"
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/loilo-inc/canarycage/mocks/mock_types"
	"github.com/loilo-inc/canarycage/types"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"
)

func TestUpgrade(t *testing.T) {
	t.Run("Upgrade", func(t *testing.T) {
		app := cli.NewApp()
		ctrl := gomock.NewController(t)
		u := mock_types.NewMockUpgrade(ctrl)
		u.EXPECT().Upgrade(gomock.Any()).Return(nil)
		app.Commands = []*cli.Command{
			commands.Upgrade(&cageapp.UpgradeCmdInput{}, func(_Ctx context.Context, _Input *cageapp.UpgradeCmdInput) (types.Upgrade, error) {
				return u, nil
			}),
		}
		err := app.Run([]string{"cage", "upgrade"})
		assert.NoError(t, err)
	})
	t.Run("Upgrade with pre-release", func(t *testing.T) {
		app := cli.NewApp()
		ctrl := gomock.NewController(t)
		u := mock_types.NewMockUpgrade(ctrl)
		u.EXPECT().Upgrade(gomock.Any()).Return(nil)
		app.Commands = []*cli.Command{
			commands.Upgrade(&cageapp.UpgradeCmdInput{PreRelease: true}, func(_Ctx context.Context, _Input *cageapp.UpgradeCmdInput) (types.Upgrade, error) {
				return u, nil
			}),
		}
		err := app.Run([]string{"cage", "upgrade", "--pre-release"})
		assert.NoError(t, err)
	})
}
