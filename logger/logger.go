package logger

import (
	"fmt"
	"io"
)

type Logger interface {
	Printf(format string, args ...any)
}

func DefaultLogger(stdout io.Writer) Logger {
	return &defaultLogger{stdout: stdout}
}

type defaultLogger struct {
	stdout io.Writer
}

func (l *defaultLogger) Printf(format string, args ...any) {
	fmt.Fprintf(l.stdout, format, args...)
}
