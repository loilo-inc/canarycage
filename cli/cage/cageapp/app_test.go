package cageapp

import (
	"testing"
	"time"
)

func TestSpinInterval(t *testing.T) {
	tests := []struct {
		name     string
		ci       bool
		expected time.Duration
	}{
		{
			name:     "CI mode returns 10 seconds",
			ci:       true,
			expected: time.Second * 10,
		},
		{
			name:     "Non-CI mode returns 100 milliseconds",
			ci:       false,
			expected: time.Millisecond * 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{CI: tt.ci}
			got := app.SpinInterval()
			if got != tt.expected {
				t.Errorf("SpinInterval() = %v, want %v", got, tt.expected)
			}
		})
	}
}
