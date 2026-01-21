package cageapp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCageCmdInput(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		input := NewCageCmdInput(strings.NewReader("test"))
		assert := assert.New(t)
		assert.NotNil(input.App)
		assert.NotNil(input.Envars)
		assert.NotNil(input.Stdin)
	})
	t.Run("with options", func(t *testing.T) {
		input := NewCageCmdInput(nil, func(c *CageCmdInput) {
			c.Envars.Region = "us-west-2"
		})
		assert := assert.New(t)
		assert.NotNil(input.App)
		assert.NotNil(input.Envars)
		assert.Equal("us-west-2", input.Envars.Region)
	})
}

func TestNewAuditCmdInput(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		input := NewAuditCmdInput()
		assert := assert.New(t)
		assert.NotNil(input.App)
	})
	t.Run("with options", func(t *testing.T) {
		input := NewAuditCmdInput(func(a *AuditCmdInput) {
			a.Region = "us-west-2"
		})
		assert := assert.New(t)
		assert.NotNil(input.App)
		assert.Equal("us-west-2", input.Region)
	})
}
