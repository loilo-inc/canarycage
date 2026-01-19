package logger_test

import (
	"bytes"
	"testing"

	"github.com/loilo-inc/canarycage/logger"
	"github.com/stretchr/testify/assert"
)

func TestDefaultLogger_Printf(t *testing.T) {
	var bin bytes.Buffer
	logger := logger.DefaultLogger(&bin)
	logger.Printf("Hello, %s!", "world")
	output := bin.String()
	assert.Equal(t, "Hello, world!", output)
}
