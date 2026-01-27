package upgrade

import (
	"context"
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/stretchr/testify/assert"
)

func TestProvideUpgradeDI(t *testing.T) {
	t.Run("successfully creates upgrade DI with valid input", func(t *testing.T) {
		input := &cageapp.UpgradeCmdInput{}

		upgrade, err := ProvideUpgradeDI(context.TODO(), input)
		assert.NoError(t, err)
		assert.NotNil(t, upgrade)
	})

	t.Run("handles nil input", func(t *testing.T) {
		upgrade, err := ProvideUpgradeDI(context.TODO(), nil)
		assert.NoError(t, err)
		assert.NotNil(t, upgrade)
	})

	t.Run("handles context", func(t *testing.T) {
		input := &cageapp.UpgradeCmdInput{}
		ctx := context.Background()

		upgrade, err := ProvideUpgradeDI(ctx, input)
		assert.NoError(t, err)
		assert.NotNil(t, upgrade)
	})
}
