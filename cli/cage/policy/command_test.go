package policy

import (
	"bytes"
	"errors"
	"testing"
)

func TestRunShortFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := NewCommand(buf, true)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("\n")) {
		t.Error("output should end with newline")
	}
}

func TestRunIndentedFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := NewCommand(buf, false)
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("\n")) {
		t.Error("output should end with newline")
	}
	if !bytes.Contains(buf.Bytes(), []byte("  ")) {
		t.Error("indented output should contain spaces")
	}
}

func TestRunWriterError(t *testing.T) {
	cmd := NewCommand(&failingWriter{}, false)
	err := cmd.Run()
	if err == nil {
		t.Error("Run() should return error when Writer fails")
	}
}

type failingWriter struct{}

func (fw *failingWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write failed")
}
