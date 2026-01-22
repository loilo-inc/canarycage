package logger

import "testing"

func TestNewSpinner(t *testing.T) {
	s := NewSpinner()
	if s == nil {
		t.Fatal("NewSpinner() returned nil")
	}
	if len(s.frames) != 10 {
		t.Errorf("expected 10 frames, got %d", len(s.frames))
	}
	if s.index != 0 {
		t.Errorf("expected initial index 0, got %d", s.index)
	}
}

func TestSpinnerNext(t *testing.T) {
	s := NewSpinner()
	expectedFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	// Test that Next() returns frames in order
	for i := 0; i < len(expectedFrames); i++ {
		frame := s.Next()
		if frame != expectedFrames[i] {
			t.Errorf("expected frame %q at index %d, got %q", expectedFrames[i], i, frame)
		}
	}

	// Test that it wraps around
	frame := s.Next()
	if frame != expectedFrames[0] {
		t.Errorf("expected frame to wrap around to %q, got %q", expectedFrames[0], frame)
	}
}
