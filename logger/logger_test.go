package logger_test

import (
	"bytes"
	"regexp"
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

		assert.Regexp(t, regexp.MustCompile(`^\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}  info  test message: hello\\n$`), stdout.String())
		assert.Equal(t, "", stderr.String())
	})

	t.Run("Errorf writes to stderr", func(t *testing.T) {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		log := logger.DefaultLogger(stdout, stderr)

		log.Errorf("error message: %d\n", 42)

		assert.Equal(t, "", stdout.String())
		assert.Regexp(t, regexp.MustCompile(`^\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}  error  error message: 42\\n$`), stderr.String())
	})

	t.Run("Printf and Errorf write to separate streams", func(t *testing.T) {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		log := logger.DefaultLogger(stdout, stderr)

		log.Printf("info\n")
		log.Errorf("error\n")

		assert.Regexp(t, regexp.MustCompile(`^\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}  info  info\\n$`), stdout.String())
		assert.Regexp(t, regexp.MustCompile(`^\\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2}  error  error\\n$`), stderr.String())
	})
}
