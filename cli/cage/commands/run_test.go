package commands_test

import (
	"fmt"
	"testing"

	"github.com/loilo-inc/canarycage/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRun(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		app, cagecli := setup(t, stdinTask)
		cagecli.EXPECT().Run(gomock.Any(), gomock.Any()).Return(&types.RunResult{}, nil)
		err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
		assert.NoError(t, err)
	})
	t.Run("basic/ci", func(t *testing.T) {
		app, cagecli := setup(t, "")
		cagecli.EXPECT().Run(gomock.Any(), gomock.Any()).Return(&types.RunResult{}, nil)
		err := app.Run([]string{"cage", "--ci", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
		assert.NoError(t, err)
	})
	t.Run("missing args", func(t *testing.T) {
		app, _ := setup(t, "")
		err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1"})
		assert.EqualError(t, err, "invalid number of arguments. expected at least 3")
	})
	t.Run("error", func(t *testing.T) {
		app, cagecli := setup(t, stdinTask)
		cagecli.EXPECT().Run(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
		err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
		assert.EqualError(t, err, "error")
	})
}
