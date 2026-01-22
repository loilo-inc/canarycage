package logger

import (
	"fmt"
)

type Color struct {
	NoColor bool
}

func (c *Color) sprintf(prefix, s, suffix string, args ...any) string {
	if c.NoColor {
		return fmt.Sprintf(s, args...)
	}
	return prefix + fmt.Sprintf(s, args...) + suffix
}

func (c *Color) Red(s string) string {
	return c.Redf("%s", s)
}
func (c *Color) Redf(s string, args ...any) string {
	return c.sprintf("\033[31m", s, "\033[0m", args...)
}

func (c *Color) Green(s string) string {
	return c.Greenf("%s", s)
}
func (c *Color) Greenf(s string, args ...any) string {
	return c.sprintf("\033[32m", s, "\033[0m", args...)
}

func (c *Color) Yellow(s string) string {
	return c.Yellowf("%s", s)
}
func (c *Color) Yellowf(s string, args ...any) string {
	return c.sprintf("\033[33m", s, "\033[0m", args...)
}

func (c *Color) Magenta(s string) string {
	return c.Magentaf("%s", s)
}
func (c *Color) Magentaf(s string, args ...any) string {
	return c.sprintf("\033[35m", s, "\033[0m", args...)
}

func (c *Color) Bold(s string) string {
	return c.Boldf("%s", s)
}
func (c *Color) Boldf(s string, args ...any) string {
	return c.sprintf("\033[1m", s, "\033[0m", args...)
}
