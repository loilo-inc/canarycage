package audit

import (
	"testing"

	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/stretchr/testify/assert"
)

func TestSeverityPrinter_Sprintf(t *testing.T) {
	tests := []struct {
		name     string
		severity ecrtypes.FindingSeverity
		format   string
		args     []any
		want     string
	}{
		{
			name:     "critical severity formats with magenta",
			severity: ecrtypes.FindingSeverityCritical,
			format:   "test %s",
			args:     []any{"critical"},
			want:     "\x1b[35mtest critical\x1b[0m",
		},
		{
			name:     "high severity formats with red",
			severity: ecrtypes.FindingSeverityHigh,
			format:   "test %s",
			args:     []any{"high"},
			want:     "\x1b[31mtest high\x1b[0m",
		},
		{
			name:     "medium severity formats with yellow",
			severity: ecrtypes.FindingSeverityMedium,
			format:   "test %s",
			args:     []any{"medium"},
			want:     "\x1b[33mtest medium\x1b[0m",
		},
		{
			name:     "low severity formats without color",
			severity: ecrtypes.FindingSeverityLow,
			format:   "test %s",
			args:     []any{"low"},
			want:     "test low",
		},
		{
			name:     "informational severity formats without color",
			severity: ecrtypes.FindingSeverityInformational,
			format:   "test %s",
			args:     []any{"info"},
			want:     "test info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &severityPrinter{
				severity: tt.severity,
			}
			got := s.Sprintf(tt.format, tt.args...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSeverityPrinter_BSprintf(t *testing.T) {
	s := &severityPrinter{
		severity: ecrtypes.FindingSeverityCritical,
	}
	got := s.BSprintf("test %s", "critical")
	want := "\x1b[1m\x1b[35mtest critical\x1b[0m\x1b[0m"
	assert.Equal(t, want, got)
}
