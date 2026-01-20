package logger

import "testing"

func TestColor_Redf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Redf("test %s", "message")
	expected := "\033[31mtest message\033[0m"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	c.NoColor = true
	result = c.Redf("test %s", "message")
	expected = "test %s"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestColor_Greenf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Greenf("test %s", "message")
	expected := "\033[32mtest message\033[0m"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	c.NoColor = true
	result = c.Greenf("test %s", "message")
	expected = "test %s"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestColor_Yellowf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Yellowf("test %s", "message")
	expected := "\033[33mtest message\033[0m"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	c.NoColor = true
	result = c.Yellowf("test %s", "message")
	expected = "test %s"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestColor_Bluef(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Bluef("test %s", "message")
	expected := "\033[34mtest message\033[0m"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	c.NoColor = true
	result = c.Bluef("test %s", "message")
	expected = "test %s"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestColor_Magentaf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Magentaf("test %s", "message")
	expected := "\033[35mtest message\033[0m"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	c.NoColor = true
	result = c.Magentaf("test %s", "message")
	expected = "test %s"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestColor_Cyanf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Cyanf("test %s", "message")
	expected := "\033[36mtest message\033[0m"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	c.NoColor = true
	result = c.Cyanf("test %s", "message")
	expected = "test %s"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestColor_Whitef(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Whitef("test %s", "message")
	expected := "\033[37mtest message\033[0m"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	c.NoColor = true
	result = c.Whitef("test %s", "message")
	expected = "test %s"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestColor_Boldf(t *testing.T) {
	c := &Color{NoColor: false}
	result := c.Boldf("test %s", "message")
	expected := "\033[1mtest message\033[0m"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	c.NoColor = true
	result = c.Boldf("test %s", "message")
	expected = "test %s"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
