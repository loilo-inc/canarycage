package logger

import "io"

type Logger interface {
	Printf(format string, args ...any)
	Errorf(format string, args ...any)
	Debugf(format string, args ...any)
}

func DefaultLogger(stdout io.Writer, stderr io.Writer) Logger {
	return newPrefixedLogger(stdout, stderr)
}
