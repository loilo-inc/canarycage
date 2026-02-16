package commands_test

import (
	"context"
	"io"
	"testing"

	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/v5/cli/cage/commands"
	"github.com/loilo-inc/canarycage/v5/mocks/mock_types"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"
)

var stdinService = "ap-notheast-1\ncluster\nservice\nyes\n"
var stdinTask = "ap-notheast-1\ncluster\nyes\n"

func setup(t *testing.T, stdin io.Reader) (*cli.App, *mock_types.MockCage) {
	ctrl := gomock.NewController(t)
	cagecli := mock_types.NewMockCage(ctrl)
	input := cageapp.NewCageCmdInput(stdin)
	app := cli.NewApp()
	cmds := commands.NewCageCommands(func(ctx context.Context, input *cageapp.CageCmdInput) (types.Cage, error) {
		return cagecli, nil
	})
	app.Commands = []*cli.Command{
		cmds.Up(input),
		cmds.RollOut(input),
		cmds.Run(input),
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "ci",
			Destination: &input.CI,
			Value:       false,
		},
	}
	return app, cagecli
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}
