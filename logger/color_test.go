package logger

import "testing"

func TestColor_Red(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Red("error")
	expected := "\033[31merror\033[0m"
	if result != expected {
		t.Errorf("Red() = %q, want %q", result, expected)
	}
}

func TestColor_Redf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Redf("error: %s", "message")
	expected := "\033[31merror: message\033[0m"
	if result != expected {
		t.Errorf("Redf() = %q, want %q", result, expected)
	}
}

func TestColor_Green(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Green("success")
	expected := "\033[32msuccess\033[0m"
	if result != expected {
		t.Errorf("Green() = %q, want %q", result, expected)
	}
}

func TestColor_Greenf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Greenf("success: %d", 100)
	expected := "\033[32msuccess: 100\033[0m"
	if result != expected {
		t.Errorf("Greenf() = %q, want %q", result, expected)
	}
}

func TestColor_Yellow(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Yellow("warning")
	expected := "\033[33mwarning\033[0m"
	if result != expected {
		t.Errorf("Yellow() = %q, want %q", result, expected)
	}
}

func TestColor_Yellowf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Yellowf("warning: %s", "test")
	expected := "\033[33mwarning: test\033[0m"
	if result != expected {
		t.Errorf("Yellowf() = %q, want %q", result, expected)
	}
}

func TestColor_Magenta(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Magenta("info")
	expected := "\033[35minfo\033[0m"
	if result != expected {
		t.Errorf("Magenta() = %q, want %q", result, expected)
	}
}

func TestColor_Magentaf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Magentaf("info: %v", true)
	expected := "\033[35minfo: true\033[0m"
	if result != expected {
		t.Errorf("Magentaf() = %q, want %q", result, expected)
	}
}

func TestColor_Bold(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Bold("bold")
	expected := "\033[1mbold\033[0m"
	if result != expected {
		t.Errorf("Bold() = %q, want %q", result, expected)
	}
}

func TestColor_Boldf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Boldf("bold: %s", "text")
	expected := "\033[1mbold: text\033[0m"
	if result != expected {
		t.Errorf("Boldf() = %q, want %q", result, expected)
	}
}

func TestColor_NoColor(t *testing.T) {
	c := &Color{NoColor: true}
	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{"Red", func() string { return c.Red("text") }, "text"},
		{"Redf", func() string { return c.Redf("text: %s", "arg") }, "text: arg"},
		{"Green", func() string { return c.Green("text") }, "text"},
		{"Yellow", func() string { return c.Yellow("text") }, "text"},
		{"Magenta", func() string { return c.Magenta("text") }, "text"},
		{"Boldf", func() string { return c.Boldf("text: %d", 42) }, "text: 42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn()
			if result != tt.expected {
				t.Errorf("%s with NoColor = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}
