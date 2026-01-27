package logger

import (
	"fmt"
	"strings"
	"time"
)

type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
	Debugf(format string, args ...any)
}

type prefixedLogger struct {
	p   Printer
	now func() time.Time
}

func DefaultLogger(p Printer) Logger {
	return &prefixedLogger{p: p, now: time.Now}
}

func (l *prefixedLogger) Errorf(format string, args ...any) {
	l.errorf("error", format, args...)
}

func (l *prefixedLogger) Debugf(format string, args ...any) {
	l.printf("debug", format, args...)
}

func (l *prefixedLogger) Infof(format string, args ...any) {
	l.printf("info", format, args...)
}

func (l *prefixedLogger) printf(level string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	prefix := fmt.Sprintf("%s  %s  ", l.now().Format("2006/01/02 15:04:05"), level)
	l.p.PrintOutf(prefix + msg)
}

func (l *prefixedLogger) errorf(level string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	prefix := fmt.Sprintf("%s  %s  ", l.now().Format("2006/01/02 15:04:05"), level)
	l.p.PrintErrf(prefix + msg)
}
