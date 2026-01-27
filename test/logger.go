package test

import (
	"fmt"
	"io"

	"github.com/loilo-inc/canarycage/logger"
)

type MockLogger struct {
	Stdout []string
	Stderr []string
	Logs   []string
}

func (m *MockLogger) Printf(format string, args ...any) {
	m.Stdout = append(m.Stdout, fmt.Sprintf(format, args...))
	m.Logs = append(m.Logs, fmt.Sprintf(format, args...))
}
func (m *MockLogger) Errorf(format string, args ...any) {
	m.Stderr = append(m.Stderr, fmt.Sprintf(format, args...))
	m.Logs = append(m.Logs, fmt.Sprintf(format, args...))
}
func (m *MockLogger) Debugf(format string, args ...any) {
	m.Stdout = append(m.Stdout, fmt.Sprintf(format, args...))
	m.Logs = append(m.Logs, fmt.Sprintf(format, args...))
}

var _ logger.Logger = (*MockLogger)(nil)

func NewLogger() logger.Logger {
	return logger.DefaultLogger(io.Discard, io.Discard)
}
