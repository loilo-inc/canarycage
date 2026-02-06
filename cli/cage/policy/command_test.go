package policy

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCommandRunPretty(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := NewCommand(buf, true)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run() error: %s", err)
	}
	var doc Document
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("invalid json output: %s", err)
	}
	if doc.Version == "" {
		t.Fatalf("expected version to be set")
	}
	if len(doc.Statement) == 0 {
		t.Fatalf("expected at least one statement")
	}
}
