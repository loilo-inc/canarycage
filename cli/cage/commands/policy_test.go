package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestPolicy(t *testing.T) {
	cmd := Policy()

	assert.Equal(t, "policy", cmd.Name)
	assert.Equal(t, "output IAM policy required for canarycage", cmd.Usage)
	assert.Len(t, cmd.Flags, 1)

	shortFlag, ok := cmd.Flags[0].(*cli.BoolFlag)
	assert.True(t, ok)
	assert.Equal(t, "short", shortFlag.Name)
	assert.Equal(t, "output short format", shortFlag.Usage)
}

func TestPolicyAction(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name: "without short flag",
			args: []string{"cage", "policy"},
		},
		{
			name: "with short flag",
			args: []string{"cage", "policy", "--short"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			app := &cli.App{
				Commands: []*cli.Command{Policy()},
				Writer:   buf,
			}
			err := app.Run(tt.args)
			assert.NoError(t, err)
		})
	}
}
