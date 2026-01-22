package logger

import (
	"fmt"
	"io"
	"log/slog"
)

type Logger interface {
	Printer
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

type Printer interface {
	Printf(format string, args ...any)
}

func DefaultLogger(stdout io.Writer) Logger {
	l := slog.New(slog.NewTextHandler(stdout, nil))
	return &defaultLogger{logger: l}
}

type defaultLogger struct {
	stdout io.Writer
	logger *slog.Logger
}

func (l *defaultLogger) Printf(format string, args ...any) {
	fmt.Fprintf(l.stdout, format, args...)
}

func (l *defaultLogger) Infof(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *defaultLogger) Errorf(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...))
}
