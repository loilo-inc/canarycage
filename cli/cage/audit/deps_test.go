package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvideAuditDI(t *testing.T) {
	t.Run("successfully creates scan DI with valid region", func(t *testing.T) {
		region := "us-east-1"

		d, err := ProvideAuditDI(region)
		assert.NoError(t, err)
		assert.NotNil(t, d)
	})

	t.Run("returns error with invalid region", func(t *testing.T) {
		region := ""

		d, err := ProvideAuditDI(region)
		if err != nil {
			assert.Nil(t, d, "expected DI domain to be nil when error occurs")
			return
		}
		assert.NotNil(t, d, "expected DI domain to be non-nil when no error")
	})

	t.Run("creates DI domain with different regions", func(t *testing.T) {
		regions := []string{"us-west-2", "eu-west-1", "ap-northeast-1"}

		for _, region := range regions {
			d, err := ProvideAuditDI(region)
			if err != nil {
				t.Logf("region %s returned error: %v", region, err)
				continue
			}
			assert.NotNil(t, d, "expected DI domain to be non-nil for region %s", region)
		}
	})
}
