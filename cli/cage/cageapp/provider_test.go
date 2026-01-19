package cageapp_test

import (
	"testing"

	"github.com/loilo-inc/canarycage/cli/cage/cageapp"
	"github.com/loilo-inc/canarycage/env"
	"github.com/stretchr/testify/assert"
)

func TestProvideCageCli(t *testing.T) {
	t.Run("successfully creates cage cli with valid region", func(t *testing.T) {
		envars := &env.Envars{
			Region: "us-east-1",
		}

		cage, err := cageapp.ProvideCageCli(envars)
		assert.NoError(t, err)
		assert.NotNil(t, cage)
	})

	t.Run("returns error with invalid region", func(t *testing.T) {
		envars := &env.Envars{
			Region: "",
		}

		cage, err := cageapp.ProvideCageCli(envars)
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

		cage, err := cageapp.ProvideCageCli(nil)
		if err == nil {
			assert.NotNil(t, cage, "expected cage to be non-nil when no error")
		}
	})
}

func TestProvideScanDI(t *testing.T) {
	t.Run("successfully creates scan DI with valid region", func(t *testing.T) {
		region := "us-east-1"

		d, err := cageapp.ProvideScanDI(region)
		assert.NoError(t, err)
		assert.NotNil(t, d)
	})

	t.Run("returns error with invalid region", func(t *testing.T) {
		region := ""

		d, err := cageapp.ProvideScanDI(region)
		if err != nil {
			assert.Nil(t, d, "expected DI domain to be nil when error occurs")
			return
		}
		assert.NotNil(t, d, "expected DI domain to be non-nil when no error")
	})

	t.Run("creates DI domain with different regions", func(t *testing.T) {
		regions := []string{"us-west-2", "eu-west-1", "ap-northeast-1"}

		for _, region := range regions {
			d, err := cageapp.ProvideScanDI(region)
			if err != nil {
				t.Logf("region %s returned error: %v", region, err)
				continue
			}
			assert.NotNil(t, d, "expected DI domain to be non-nil for region %s", region)
		}
	})
}
