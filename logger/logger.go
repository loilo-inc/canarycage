package logger

import (
	"fmt"
	"io"
)

type Logger interface {
	Printf(format string, args ...any)
	Errorf(format string, args ...any)
}

func DefaultLogger(stdout io.Writer, stderr io.Writer) Logger {
	return &defaultLogger{stdout: stdout, stderr: stderr}
}

type defaultLogger struct {
	stdout io.Writer
	stderr io.Writer
}

func (l *defaultLogger) Printf(format string, args ...any) {
	fmt.Fprintf(l.stdout, format, args...)
}

func (l *defaultLogger) Errorf(format string, args ...any) {
	fmt.Fprintf(l.stderr, format, args...)
}
