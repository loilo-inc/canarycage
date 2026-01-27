package logger

import (
	"bytes"
	"testing"
)

func TestNewPrinter(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	p := NewPrinter(stdout, stderr)

	if p == nil {
		t.Fatal("NewPrinter returned nil")
	}

	// Verify the printer implements the interface
	var _ Printer = p

	// Test that PrintOutf writes to stdout
	p.PrintOutf("test stdout")
	if stdout.String() != "test stdout" {
		t.Errorf("PrintOutf: got %q, want %q", stdout.String(), "test stdout")
	}

	// Test that PrintErrf writes to stderr
	p.PrintErrf("test stderr")
	if stderr.String() != "test stderr" {
		t.Errorf("PrintErrf: got %q, want %q", stderr.String(), "test stderr")
	}
}
