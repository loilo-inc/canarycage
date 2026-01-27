package logger_test

import (
	"bytes"
	"testing"

	"github.com/loilo-inc/canarycage/logger"
	"github.com/stretchr/testify/assert"
)

func TestDefaultLogger(t *testing.T) {
	t.Run("Printf writes to stdout", func(t *testing.T) {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		log := logger.DefaultLogger(stdout, stderr)

		log.Printf("test message: %s\n", "hello")

		assert.Equal(t, "test message: hello\n", stdout.String())
		assert.Equal(t, "", stderr.String())
	})

	t.Run("Errorf writes to stderr", func(t *testing.T) {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		log := logger.DefaultLogger(stdout, stderr)

		log.Errorf("error message: %d\n", 42)

		assert.Equal(t, "", stdout.String())
		assert.Equal(t, "error message: 42\n", stderr.String())
	})

	t.Run("Printf and Errorf write to separate streams", func(t *testing.T) {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		log := logger.DefaultLogger(stdout, stderr)

		log.Printf("info\n")
		log.Errorf("error\n")

		assert.Equal(t, "info\n", stdout.String())
		assert.Equal(t, "error\n", stderr.String())
	})
}
