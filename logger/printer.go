package logger

import (
	"fmt"
	"io"
)

type Printer interface {
	PrintOutf(format string, args ...any)
	PrintErrf(format string, args ...any)
}

type printer struct {
	stdout io.Writer
	stderr io.Writer
}

var _ Printer = (*printer)(nil)

func NewPrinter(stdout io.Writer, stderr io.Writer) Printer {
	return &printer{stdout: stdout, stderr: stderr}
}

func (p *printer) PrintOutf(format string, args ...any) {
	fmt.Fprintf(p.stdout, format, args...)
}

func (p *printer) PrintErrf(format string, args ...any) {
	fmt.Fprintf(p.stderr, format, args...)
}
