package cageapp

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	assert := assert.New(t)

	assert.NotNil(app, "NewApp() returned nil")
	assert.Equal(os.Stdin, app.Stdin, "expected Stdin to be os.Stdin")
	assert.False(app.CI, "expected CI to be false")
}
