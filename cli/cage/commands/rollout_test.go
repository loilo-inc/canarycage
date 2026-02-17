package commands_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRollOut(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		app, cagecli := setup(t, strings.NewReader(stdinService))
		cagecli.EXPECT().RollOut(gomock.Any(), &types.RollOutInput{}).Return(&types.RollOutResult{}, nil)
		err := app.Run(context.Background(), []string{"cage", "rollout", "--region", "ap-notheast-1", "../../../fixtures"})
		assert.NoError(t, err)
	})
	t.Run("basic/ci", func(t *testing.T) {
		app, cagecli := setup(t, nil)
		cagecli.EXPECT().RollOut(gomock.Any(), &types.RollOutInput{}).Return(&types.RollOutResult{}, nil)
		err := app.Run(context.Background(), []string{"cage", "--ci", "rollout", "--region", "ap-notheast-1", "../../../fixtures"})
		assert.NoError(t, err)
	})
	t.Run("basic/update-service", func(t *testing.T) {
		app, cagecli := setup(t, strings.NewReader(stdinService))
		cagecli.EXPECT().RollOut(gomock.Any(), &types.RollOutInput{UpdateService: true}).Return(&types.RollOutResult{}, nil)
		err := app.Run(context.Background(), []string{"cage", "rollout", "--region", "ap-notheast-1", "--updateService", "../../../fixtures"})
		assert.NoError(t, err)
	})
	t.Run("missing args", func(t *testing.T) {
		app, _ := setup(t, nil)
		err := app.Run(context.Background(), []string{"cage", "rollout", "--region", "ap-notheast-1"})
		assert.EqualError(t, err, "invalid number of arguments. expected at least 1")
	})
	t.Run("reading stdin error", func(t *testing.T) {
		app, _ := setup(t, &errorReader{})
		err := app.Run(context.Background(), []string{"cage", "rollout", "--region", "ap-notheast-1", "../../../fixtures"})
		assert.EqualError(t, err, "failed to read from stdin: EOF")
	})
	t.Run("error", func(t *testing.T) {
		app, cagecli := setup(t, strings.NewReader(stdinService))
		cagecli.EXPECT().RollOut(gomock.Any(), &types.RollOutInput{}).Return(&types.RollOutResult{}, fmt.Errorf("error"))
		err := app.Run(context.Background(), []string{"cage", "rollout", "--region", "ap-notheast-1", "../../../fixtures"})
		assert.EqualError(t, err, "error")
	})
}
