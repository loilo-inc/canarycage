package cageapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCageCmdInput(t *testing.T) {
	input := NewCageCmdInput(nil, func(c *CageCmdInput) {
		c.Envars.Region = "us-west-2"
	})
	assert := assert.New(t)
	assert.NotNil(input.App)
	assert.NotNil(input.Envars)
	assert.Equal("us-west-2", input.Envars.Region)
}

func TestNewAuditCmdInput(t *testing.T) {
	input := NewAuditCmdInput(func(a *AuditCmdInput) {
		a.Region = "us-west-2"
	})
	assert := assert.New(t)
	assert.NotNil(input.App)
	assert.Equal("us-west-2", input.Region)
}

func TestNewUpgradeCmdInput(t *testing.T) {
	input := NewUpgradeCmdInput(func(a *UpgradeCmdInput) {
		a.NoColor = true
	})
	assert := assert.New(t)
	assert.NotNil(input.App)
	assert.Equal(true, input.NoColor)
}
