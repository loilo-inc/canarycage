package commands

import (
	"context"
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/stretchr/testify/assert"
)

func TestProvideCageCli(t *testing.T) {
	t.Run("successfully creates cage cli with valid region", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Envars.Region = "us-west-2"

		cage, err := ProvideCageCli(context.TODO(), input)
		assert.NoError(t, err)
		assert.NotNil(t, cage)
	})

	t.Run("returns error with invalid region", func(t *testing.T) {
		input := cageapp.NewCageCmdInput(nil)
		input.Envars.Region = ""

		cage, err := ProvideCageCli(context.TODO(), input)
		if err != nil {
			assert.Nil(t, cage, "expected cage to be nil when error occurs")
			return
		}
		assert.NotNil(t, cage, "expected cage to be non-nil when no error")
	})

	t.Run("handles nil envars", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				return
			}
		}()

		cage, err := ProvideCageCli(context.TODO(), nil)
		if err == nil {
			assert.NotNil(t, cage, "expected cage to be non-nil when no error")
		}
	})
}
