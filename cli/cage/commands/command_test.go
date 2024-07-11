package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/loilo-inc/canarycage/env"
	"github.com/loilo-inc/canarycage/mocks/mock_types"
	"github.com/loilo-inc/canarycage/test"
	"github.com/loilo-inc/canarycage/types"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"
)

func TestCommands(t *testing.T) {
	region := "ap-notheast-1"
	cluster := "cluster"
	service := "service"
	stdinService := fmt.Sprintf("%s\n%s\n%s\n%s\n", region, cluster, service, "yes")
	stdinTask := fmt.Sprintf("%s\n%s\n%s\n", region, cluster, "yes")
	setup := func(t *testing.T, input string) (*cli.App, *mock_types.MockCage) {
		ctrl := gomock.NewController(t)
		stdin := strings.NewReader(input)
		cagecli := mock_types.NewMockCage(ctrl)
		app := cli.NewApp()
		cmds := NewCageCommands(stdin, func(envars *env.Envars) (types.Cage, error) {
			return cagecli, nil
		})
		envars := env.Envars{CI: input == ""}
		app.Commands = []*cli.Command{
			cmds.Up(&envars),
			cmds.RollOut(&envars),
			cmds.Run(&envars),
		}
		return app, cagecli
	}
	t.Run("rollout", func(t *testing.T) {
		t.Run("basic", func(t *testing.T) {
			app, cagecli := setup(t, stdinService)
			cagecli.EXPECT().RollOut(gomock.Any(), &types.RollOutInput{}).Return(&types.RollOutResult{}, nil)
			err := app.Run([]string{"cage", "rollout", "--region", "ap-notheast-1", "../../../fixtures"})
			assert.NoError(t, err)
		})
		t.Run("basic/ci", func(t *testing.T) {
			app, cagecli := setup(t, "")
			cagecli.EXPECT().RollOut(gomock.Any(), &types.RollOutInput{}).Return(&types.RollOutResult{}, nil)
			err := app.Run([]string{"cage", "rollout", "--region", "ap-notheast-1", "../../../fixtures"})
			assert.NoError(t, err)
		})
		t.Run("basic/udate-service", func(t *testing.T) {
			app, cagecli := setup(t, stdinService)
			cagecli.EXPECT().RollOut(gomock.Any(), &types.RollOutInput{UpdateService: true}).Return(&types.RollOutResult{}, nil)
			err := app.Run([]string{"cage", "rollout", "--region", "ap-notheast-1", "--updateService", "../../../fixtures"})
			assert.NoError(t, err)
		})
		t.Run("error", func(t *testing.T) {
			app, cagecli := setup(t, stdinService)
			cagecli.EXPECT().RollOut(gomock.Any(), &types.RollOutInput{}).Return(&types.RollOutResult{}, fmt.Errorf("error"))
			err := app.Run([]string{"cage", "rollout", "--region", "ap-notheast-1", "../../../fixtures"})
			assert.EqualError(t, err, "error")
		})
	})
	t.Run("up", func(t *testing.T) {
		t.Run("basic", func(t *testing.T) {
			app, cagecli := setup(t, stdinService)
			cagecli.EXPECT().Up(gomock.Any()).Return(&types.UpResult{}, nil)
			err := app.Run([]string{"cage", "up", "--region", "ap-notheast-1", "../../../fixtures"})
			assert.NoError(t, err)
		})
		t.Run("basic/ci", func(t *testing.T) {
			app, cagecli := setup(t, "")
			cagecli.EXPECT().Up(gomock.Any()).Return(&types.UpResult{}, nil)
			err := app.Run([]string{"cage", "up", "--region", "ap-notheast-1", "../../../fixtures"})
			assert.NoError(t, err)
		})
		t.Run("error", func(t *testing.T) {
			app, cagecli := setup(t, stdinService)
			cagecli.EXPECT().Up(gomock.Any()).Return(nil, fmt.Errorf("error"))
			err := app.Run([]string{"cage", "up", "--region", "ap-notheast-1", "../../../fixtures"})
			assert.EqualError(t, err, "error")
		})
	})
	t.Run("run", func(t *testing.T) {
		t.Run("basic", func(t *testing.T) {
			app, cagecli := setup(t, stdinTask)
			cagecli.EXPECT().Run(gomock.Any(), gomock.Any()).Return(&types.RunResult{}, nil)
			err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
			assert.NoError(t, err)
		})
		t.Run("basic/ci", func(t *testing.T) {
			app, cagecli := setup(t, "")
			cagecli.EXPECT().Run(gomock.Any(), gomock.Any()).Return(&types.RunResult{}, nil)
			err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
			assert.NoError(t, err)
		})
		t.Run("error", func(t *testing.T) {
			app, cagecli := setup(t, stdinTask)
			cagecli.EXPECT().Run(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
			err := app.Run([]string{"cage", "run", "--region", "ap-notheast-1", "../../../fixtures", "container", "exec"})
			assert.EqualError(t, err, "error")
		})
	})
}

func TestSetupCage(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		envars := &env.Envars{Region: "us-west-2"}
		cageCli := mock_types.NewMockCage(gomock.NewController(t))
		cmd := NewCageCommands(nil, func(envars *env.Envars) (types.Cage, error) {
			return cageCli, nil
		})
		v, err := cmd.setupCage(envars, "../../../fixtures")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, v, cageCli)
		assert.Equal(t, envars.Service, "service")
		assert.Equal(t, envars.Cluster, "cluster")
		assert.NotNil(t, envars.ServiceDefinitionInput)
		assert.NotNil(t, envars.TaskDefinitionInput)
	})
	t.Run("should skip load task definition if --taskDefinitionArn provided", func(t *testing.T) {
		envars := &env.Envars{Region: "us-west-2", TaskDefinitionArn: "arn"}
		cageCli := mock_types.NewMockCage(gomock.NewController(t))
		cmd := NewCageCommands(nil, func(envars *env.Envars) (types.Cage, error) {
			return cageCli, nil
		})
		v, err := cmd.setupCage(envars, "../../../fixtures")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, v, cageCli)
		assert.Equal(t, envars.Service, "service")
		assert.Equal(t, envars.Cluster, "cluster")
		assert.NotNil(t, envars.ServiceDefinitionInput)
		assert.Nil(t, envars.TaskDefinitionInput)
	})
	t.Run("should error if error returned from NewCage", func(t *testing.T) {
		envars := &env.Envars{Region: "us-west-2"}
		cmd := NewCageCommands(nil, func(envars *env.Envars) (types.Cage, error) {
			return nil, test.Err
		})
		_, err := cmd.setupCage(envars, "../../../fixtures")
		assert.EqualError(t, err, "error")
	})
}
