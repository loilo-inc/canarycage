package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type prefixedLogger struct {
	stdout io.Writer
	stderr io.Writer
	now    func() time.Time
}

func newPrefixedLogger(stdout, stderr io.Writer) *prefixedLogger {
	return &prefixedLogger{stdout: stdout, stderr: stderr, now: time.Now}
}

func (l *prefixedLogger) Printf(format string, args ...any) {
	l.printf("info", format, args...)
}

func (l *prefixedLogger) Errorf(format string, args ...any) {
	l.errorf("error", format, args...)
}

func (l *prefixedLogger) Debugf(format string, args ...any) {
	l.printf("debug", format, args...)
}

func (l *prefixedLogger) printf(level string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	prefix := fmt.Sprintf("%s  %s  ", l.now().Format("2006/01/02 15:04:05"), level)
	fmt.Fprint(l.stdout, prefix+msg)
}

func (l *prefixedLogger) errorf(level string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	prefix := fmt.Sprintf("%s  %s  ", l.now().Format("2006/01/02 15:04:05"), level)
	fmt.Fprint(l.stderr, prefix+msg)
}

var std = newPrefixedLogger(os.Stdout, os.Stderr)

// Printf logs an info-level message with a timestamp prefix.
func Printf(format string, args ...any) {
	std.printf("info", format, args...)
}

// Errorf logs an error-level message with a timestamp prefix.
func Errorf(format string, args ...any) {
	std.errorf("error", format, args...)
}

// Debugf logs a debug-level message with a timestamp prefix.
func Debugf(format string, args ...any) {
	std.printf("debug", format, args...)
}
