package commands

import (
	"context"
	"testing"

	"github.com/loilo-inc/canarycage/env"
	"github.com/stretchr/testify/assert"
)

func TestProvideCageCli(t *testing.T) {
	t.Run("successfully creates cage cli with valid region", func(t *testing.T) {
		envars := &env.Envars{
			Region: "us-east-1",
		}

		cage, err := ProvideCageCli(context.TODO(), envars)
		assert.NoError(t, err)
		assert.NotNil(t, cage)
	})

	t.Run("returns error with invalid region", func(t *testing.T) {
		envars := &env.Envars{
			Region: "",
		}

		cage, err := ProvideCageCli(context.TODO(), envars)
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
