package logger

import (
	"strings"
	"testing"
	"time"
)

type mockPrinter struct {
	outMessages []string
	errMessages []string
}

func (m *mockPrinter) Printf(format string, args ...any) {
	m.outMessages = append(m.outMessages, format)
}

func (m *mockPrinter) PrintErrf(format string, args ...any) {
	m.errMessages = append(m.errMessages, format)
}

func TestDefaultLogger(t *testing.T) {
	p := &mockPrinter{}
	logger := DefaultLogger(p)
	if logger == nil {
		t.Fatal("DefaultLogger returned nil")
	}
}

func TestPrefixedLogger_Infof(t *testing.T) {
	p := &mockPrinter{}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	logger := &prefixedLogger{
		p:   p,
		now: func() time.Time { return fixedTime },
	}

	logger.Infof("test message")

	if len(p.outMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(p.outMessages))
	}
	expected := "2024/01/01 12:00:00  info  test message\n"
	if p.outMessages[0] != expected {
		t.Errorf("expected %q, got %q", expected, p.outMessages[0])
	}
}

func TestPrefixedLogger_Debugf(t *testing.T) {
	p := &mockPrinter{}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	logger := &prefixedLogger{
		p:   p,
		now: func() time.Time { return fixedTime },
	}

	logger.Debugf("debug info")

	if len(p.outMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(p.outMessages))
	}
	expected := "2024/01/01 12:00:00  debug  debug info\n"
	if p.outMessages[0] != expected {
		t.Errorf("expected %q, got %q", expected, p.outMessages[0])
	}
}

func TestPrefixedLogger_Errorf(t *testing.T) {
	p := &mockPrinter{}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	logger := &prefixedLogger{
		p:   p,
		now: func() time.Time { return fixedTime },
	}

	logger.Errorf("error occurred")

	if len(p.errMessages) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(p.errMessages))
	}
	expected := "2024/01/01 12:00:00  error  error occurred\n"
	if p.errMessages[0] != expected {
		t.Errorf("expected %q, got %q", expected, p.errMessages[0])
	}
}

func TestPrefixedLogger_WithFormatArgs(t *testing.T) {
	p := &mockPrinter{}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	logger := &prefixedLogger{
		p:   p,
		now: func() time.Time { return fixedTime },
	}

	logger.Infof("user %s logged in with id %d", "alice", 123)

	expected := "2024/01/01 12:00:00  info  user alice logged in with id 123\n"
	if p.outMessages[0] != expected {
		t.Errorf("expected %q, got %q", expected, p.outMessages[0])
	}
}

func TestPrefixedLogger_MessageWithNewline(t *testing.T) {
	p := &mockPrinter{}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	logger := &prefixedLogger{
		p:   p,
		now: func() time.Time { return fixedTime },
	}

	logger.Infof("message with newline\n")

	if strings.Count(p.outMessages[0], "\n") != 1 {
		t.Errorf("expected single newline, got: %q", p.outMessages[0])
	}
}
