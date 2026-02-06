package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultDocumentIncludesActions(t *testing.T) {
	doc := DefaultDocument()
	assert.Equal(t, "2012-10-17", doc.Version)
	if assert.Len(t, doc.Statement, len(serviceSpecs)) {
		for _, stmt := range doc.Statement {
			assert.Equal(t, "Allow", stmt.Effect)
			assert.Equal(t, "*", stmt.Resource)
			assert.NotEmpty(t, stmt.Action)
		}
	}
}
