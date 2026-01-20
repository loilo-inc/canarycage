package logger

import "fmt"

type Color struct {
	NoColor bool
}

func (c *Color) Redf(s string, args ...interface{}) string {
	if c.NoColor {
		return s
	}
	return "\033[31m" + fmt.Sprintf(s, args...) + "\033[0m"
}

func (c *Color) Greenf(s string, args ...interface{}) string {
	if c.NoColor {
		return s
	}
	return "\033[32m" + fmt.Sprintf(s, args...) + "\033[0m"
}

func (c *Color) Yellowf(s string, args ...interface{}) string {
	if c.NoColor {
		return s
	}
	return "\033[33m" + fmt.Sprintf(s, args...) + "\033[0m"
}
func (c *Color) Bluef(s string, args ...interface{}) string {
	if c.NoColor {
		return s
	}
	return "\033[34m" + fmt.Sprintf(s, args...) + "\033[0m"
}
func (c *Color) Magentaf(s string, args ...interface{}) string {
	if c.NoColor {
		return s
	}
	return "\033[35m" + fmt.Sprintf(s, args...) + "\033[0m"
}
func (c *Color) Cyanf(s string, args ...interface{}) string {
	if c.NoColor {
		return s
	}
	return "\033[36m" + fmt.Sprintf(s, args...) + "\033[0m"
}
func (c *Color) Whitef(s string, args ...interface{}) string {
	if c.NoColor {
		return s
	}
	return "\033[37m" + fmt.Sprintf(s, args...) + "\033[0m"
}

func (c *Color) Boldf(s string, args ...interface{}) string {
	if c.NoColor {
		return s
	}
	return "\033[1m" + fmt.Sprintf(s, args...) + "\033[0m"
}
