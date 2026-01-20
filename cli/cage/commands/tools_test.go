package commands_test

import (
	"io"
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/cli/cage/commands"
	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/mocks/mock_types"
	"github.com/loilo-inc/canarycage/types"
	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"
)

var stdinService = "ap-notheast-1\ncluster\nservice\nyes\n"
var stdinTask = "ap-notheast-1\ncluster\nyes\n"

func setup(t *testing.T, input io.Reader) (*cli.App, *mock_types.MockCage) {
	ctrl := gomock.NewController(t)
	cagecli := mock_types.NewMockCage(ctrl)
	cageapp := &cageapp.App{Stdin: input}
	app := cli.NewApp()
	cmds := commands.NewCageCommands(func(envars *env.Envars) (types.Cage, error) {
		return cagecli, nil
	})
	app.Commands = []*cli.Command{
		cmds.Up(cageapp),
		cmds.RollOut(cageapp),
		cmds.Run(cageapp),
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "ci",
			Destination: &cageapp.CI,
			Value:       false,
		},
	}
	return app, cagecli
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}
