package commands_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRun(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		app, cagecli := setup(t, strings.NewReader(stdinTask))
		cagecli.EXPECT().Run(gomock.Any(), gomock.Any()).Return(&types.RunResult{}, nil)
		err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
		assert.NoError(t, err)
	})
	t.Run("basic/ci", func(t *testing.T) {
		app, cagecli := setup(t, nil)
		cagecli.EXPECT().Run(gomock.Any(), gomock.Any()).Return(&types.RunResult{}, nil)
		err := app.Run([]string{"cage", "--ci", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
		assert.NoError(t, err)
	})
	t.Run("missing args", func(t *testing.T) {
		app, _ := setup(t, nil)
		err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1"})
		assert.EqualError(t, err, "invalid number of arguments. expected at least 3")
	})
	t.Run("reading stdin error", func(t *testing.T) {
		app, _ := setup(t, &errorReader{})
		err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
		assert.EqualError(t, err, "failed to read from stdin: EOF")
	})
	t.Run("error", func(t *testing.T) {
		app, cagecli := setup(t, strings.NewReader(stdinTask))
		cagecli.EXPECT().Run(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
		err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
		assert.EqualError(t, err, "error")
	})
}
