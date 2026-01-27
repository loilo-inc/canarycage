package test

import (
	"fmt"

	"github.com/loilo-inc/canarycage/logger"
)

type MockPrinter struct {
	Stdout []string
	Stderr []string
	Logs   []string
}

func (m *MockPrinter) PrintOutf(format string, args ...any) {
	m.Stdout = append(m.Stdout, fmt.Sprintf(format, args...))
	m.Logs = append(m.Logs, fmt.Sprintf(format, args...))
}

func (m *MockPrinter) PrintErrf(format string, args ...any) {
	m.Stderr = append(m.Stderr, fmt.Sprintf(format, args...))
	m.Logs = append(m.Logs, fmt.Sprintf(format, args...))
}

var _ logger.Printer = (*MockPrinter)(nil)

func NewMockPrinter() *MockPrinter {
	return &MockPrinter{}
}

func NewLogger() logger.Logger {
	return logger.DefaultLogger(NewMockPrinter())
}
